#!/usr/bin/env perl 
use strict;
use warnings;
use Getopt::Std;

my %opts;
getopts "ihf",\%opts;


my $mirror;
if ( $opts{h} ) {
	$mirror = "http://mirrors.aliyun.com/deepin";
}
elsif ( $opts{f} ) {
	$mirror = "ftp://mirrors.xmu.edu.cn/deepin/deepin";
}
else {
	die "unknown option\n";
}

my $amd64_package_url = "$mirror/dists/trusty/main/binary-amd64/Packages.gz";
system "wget '$amd64_package_url' -O /tmp/Package.gz";
print qx(gzip -kfvd /tmp/Package.gz);


my $fpath = "/tmp/Package";
open PACKAGE , "<" ,$fpath or die "can't open file `$fpath` $!";

my $taskname = "my perl test";
my ($url,$size,$md5);
my (@urls, @sizes, @md5s );

while (<PACKAGE>){
	if (/^Filename: (.+)\n$/){
		$url = "$mirror/$1";
		$size = <PACKAGE>;

		if ( $size =~ /^Size: (.+)\n$/ ) {
			$size = $1;
		} else {
			die " size field not found.";
		}
		<PACKAGE>; #SHA256
		<PACKAGE>; #SHA1
		$md5 = <PACKAGE>;
		if ( $md5 =~ /^MD5sum: (.+)\n$/ ) {
			$md5 = $1;
		} else {
			die "md5sum field not found.";
		}
		
		print "$url|$size|$md5\n";
	}

}
close PACKAGE;

