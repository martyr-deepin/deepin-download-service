#!/usr/bin/env perl 
use strict;
use warnings;
use Getopt::Std;
use Net::DBus;
use Net::DBus::Reactor;
my $bus = Net::DBus->system;

my $obj_path = "/com/deepin/download/service";
my $interfaces = "com.deepin.download.service";

my $dds = $bus->get_service("com.deepin.download.service");

my $object = $dds->get_object($obj_path, $interfaces );

my %opts;
getopts('lp:a:s:r:',\%opts);
# p pause
# a add
# s stop
# r resume
# l list

my $taskname = "my perl test";
qx(mkdir -p  "/tmp/dds/");
my $store_dir = "/tmp/dds";
my ($url,$size,$md5);
my (@urls, @sizes, @md5s );


if ( $opts{a} ){
	print "add\n";
	open INPUT, '<',$opts{a}
		or die "can't open file `$opts{a}` $! ";
	while ( <INPUT> ){
		chomp;
		($url, $size, $md5) =split /\|/;
		push @urls, $url;
		push @sizes, $size;
		push @md5s, $md5;
	}
	close INPUT;
	my $taskid = $object->AddTask($taskname, \@urls, \@sizes, \@md5s, $store_dir );
	print $taskid . "\n";
} 
elsif ( $opts{l}) {
	print "list\n";	

	my %task_ids;
	$object->connect_to_signal( "Update", sub {
		my ($taskid, $progress, $speed, $finish,$total,$downloadSize, $totalSize ) = @_;
		$task_ids{ $taskid } = 1;
		} );

	my $reactor = Net::DBus::Reactor->main();

	$reactor->add_timeout(1000, sub {
			print join "\n", keys %task_ids,'';
			$reactor->shutdown 
		} );
	$reactor->run;
}
elsif ( $opts{p} ){
	print "pause\n";
	$object->PauseTask( $opts{p} );
}
elsif ( $opts{r} ){
	print "resume\n";
	$object->ResumeTask( $opts{r} );
}
elsif ( $opts{s} ){
	print "stop\n";
	$object->StopTask( $opts{s} );
}
