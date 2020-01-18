#!/bin/sh

brbundle embedded -p docradle -o ../configschema_gen.go ../cue/
gocredits .. > ../CREDITS.txt
