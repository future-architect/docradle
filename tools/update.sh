#!/bin/sh

brbundle embedded -p docradle -o ../configschema_gen.go ../data/
gocredits .. > ../CREDITS.txt
