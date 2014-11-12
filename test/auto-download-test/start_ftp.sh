#!/bin/bash

#perl ./dds_watch_cui.pl
tmpdir=/tmp/dds/
rm -fr ${tmpdir}/*
mkdir -p ${tmpdir} || true

pkglist=${tmpdir}ftp_list
perl ./package_get.pl -f > ${pkglist}

split -l 16 -d ${pkglist} ${tmpdir}ftp_split_list
total=$(find /tmp/dds/ -name ftp_split_* | wc -l)
RANDOM=$(od -An -N2 -i /dev/random)
select=$(expr $RANDOM \% $total)
array_list=($(find /tmp/dds/ -name ftp_split_*))
echo Download list ${array_list[$select]}

perl ./dds_ctrl.pl -a ${array_list[$select]}

