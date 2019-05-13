#
#  Copyright 2018 Nalej
# 

include scripts/Makefile.golang
include scripts/Makefile.vagrant

.DEFAULT_GOAL := all

# Name of the target applications to be built
APPS=edge-controller
