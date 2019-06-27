/*
 * Copyright (C) 2018 Nalej - All Rights Reserved
 */

package connection

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/rs/zerolog/log"
)

// SSHType defines the type of connection.
const SSHType ConnectionType = "ssh"

// SSHConnection structure with the information required to establish an SSH connection to a remote host.
type SSHConnection struct {
	Type     ConnectionType `json:"type"` // Needed for proper serialization
	Address  string         `json:"address"`
	Port     string         `json:"port,omitempty"`
	Username string         `json:"username"`
	Password string         `json:"password,omitempty"`
	// TODO: [DP-245] Embed actual key
	KeyFile string `json:"keyfile,omitempty"`
	// Private key should contain the string representation of .ssh/id_rsa
	PrivateKey string `json:"privateKey"`
}

// GetSSHConfig returns the connection configuration.
func (conn *SSHConnection) GetSSHConfig() (*ssh.ClientConfig, error) {
	var (
		sshConfig *ssh.ClientConfig = nil

		useKey bool = false
		key    []byte
		signer ssh.Signer

		err error
	)

	// use key file if available
	if conn.PrivateKey == "" && conn.KeyFile != "" {
		key, err = ioutil.ReadFile(conn.KeyFile)
		if err != nil {
			log.Error().Str("file", conn.KeyFile).Msg("Unable to open key file")
		} else {
			conn.PrivateKey = string(key)
		}
	}

	// Either loaded from file or already specified.
	useKey = conn.PrivateKey != ""

	// Read key file successfully
	if useKey == true {
		var pemBytes = []byte(conn.PrivateKey)
		signer, err = ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			log.Error().Str("key", conn.PrivateKey).Msg("Unable to parse key")
			useKey = false
		}
	}

	sshConfig = &ssh.ClientConfig{
		User: conn.Username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: time.Second * 15,
	}

	// Parsed key successfully
	if useKey == true {
		sshConfig.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if conn.Password != "" {
		sshConfig.Auth = []ssh.AuthMethod{ssh.Password(conn.Password)}
	} else {
		return nil, errors.New("no authentication method found")
	}

	return sshConfig, nil
}

func (conn *SSHConnection) createClient() (*ssh.Client, error) {
	sshConfig, err := conn.GetSSHConfig()
	if err != nil {
		return nil, err
	}

	sshAddress := fmt.Sprintf("%s:%s", conn.Address, conn.Port)
	client, err := ssh.Dial("tcp", sshAddress, sshConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// OpenSession generates a new session.
func (conn *SSHConnection) OpenSession() (*ssh.Client, *ssh.Session, error) {
	client, err := conn.createClient()
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}

// Execute a given command.
func (conn *SSHConnection) Execute(command string) ([]byte, error) {
	client, session, err := conn.OpenSession()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	defer session.Close()

	var stderrBuffer bytes.Buffer
	stderrReader, err := session.StderrPipe()
	if err != nil {
		return nil, err
	}
	go io.Copy(&stderrBuffer, stderrReader)

	log.Debug().Str("command", command).Msg("Executing command")
	output, err := session.Output(command)
	if err != nil {
		err = fmt.Errorf("Error executing %s, error: %v.\nSTDOUT\n%sSTDERR\n%s",
			command, err, output, stderrBuffer.Bytes())
	}

	return output, err
}

// Copy a file to a remote host or viceversa.
func (conn *SSHConnection) Copy(lpath, rpath string, remoteSource bool) error {
	client, session, err := conn.OpenSession()
	if err != nil {
		return err
	}
	defer client.Close()
	defer session.Close()

	// rpath -> lpath
	if remoteSource == true {
		r, err := session.StdoutPipe()
		if err != nil {
			return err
		}
		file, err := os.OpenFile(lpath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		log.Info().Str("rpath", rpath).Msg("Transferring file")

		if err := session.Start("cat " + rpath); err != nil {
			log.Error().Err(err).Msg("Reading remote file failed")
			return err
		}
		_, err = io.Copy(file, r)
		if err != nil {
			return err
		}
		if err := session.Wait(); err != nil {
			return err
		}
		return nil
	}

	// lpath -> rpath
	// Create session pipe to send data over.
	w, err := session.StdinPipe()
	if err != nil {
		log.Error().Msg("Error creating ssh session pipe")
		return err
	}
	defer w.Close()

	// Open local file
	src, err := os.Open(lpath)
	if err != nil {
		log.Error().Str("lpath", lpath).Msg("Error opening")
		return err
	}
	defer src.Close()

	// Stat local file
	srcstat, err := src.Stat()
	if err != nil {
		log.Error().Str("lpath", lpath).Msg("Error getting file size")
		return err
	}

	// Start SCP command. This will make the remote host wait for a file
	// through the SCP protocol on stdin.
	if err := session.Start(fmt.Sprintf("scp -t %s", rpath)); err != nil {
		return err
	}

	// Start transfer
	fmt.Fprintln(w, "C0644", srcstat.Size(), filepath.Base(lpath))
	log.Info().Int64("size", srcstat.Size()).Str("address", conn.Address).Msg("Transferring file")
	if srcstat.Size() > 0 {
		written, err := io.Copy(w, src)
		log.Debug().Int64("written", written).Str("address", conn.Address).Msg("Bytes written")
		if err != nil {
			return err
		}
	}

	// End of command
	fmt.Fprintf(w, "\x00")
	w.Close()

	// Waits until transfer is done.
	if err := session.Wait(); err != nil {
		return err
	}

	return nil
}

// IsOnline checks the connectivity. We actually set up a connection to check connectivity,
// so we also know the authentication mechanism works. Although there is a timeout, it can
// still take a long time to do this one by one on a large cluster, so it's likely a good
// idea to do this in parallel, but not too much in parallel :)
func (conn *SSHConnection) IsOnline() (bool, error) {
	client, err := conn.createClient()
	if err != nil {
		return false, err
	}

	// We don't actually need the client, just checking connectivity
	client.Close()

	return true, nil
}

// NewSSHConnection creates a new SSHConnection structure.
func NewSSHConnection(address, port, username, password, privateKeyFile string, privateKey string) (*SSHConnection, error) {
	if address == "" {
		return nil, errors.New("ssh connection needs address")
	}

	if port == "" {
		// Default port
		port = "22"
	}

	if username == "" {
		return nil, errors.New("ssh connection needs username")
	}

	if password == "" && privateKeyFile == "" && privateKey == "" {
		return nil, errors.New("ssh connection needs password or key file")
	}

	sshConnection := &SSHConnection{
		Type:       SSHType,
		Address:    address,
		Port:       port,
		Username:   username,
		Password:   password,
		KeyFile:    privateKeyFile,
		PrivateKey: privateKey,
	}

	return sshConnection, nil
}

// NewEmptySSHConnection creates an empty SSH connection.
func NewEmptySSHConnection() Connection {
	conn := &SSHConnection{}
	return Connection(conn)
}

func init() {
	AddConnectionType(SSHType, NewEmptySSHConnection)
}
