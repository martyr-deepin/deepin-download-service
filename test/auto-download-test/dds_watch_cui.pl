#!/usr/bin/env perl 
use v5.18;
use warnings;

use Net::DBus;
use Net::DBus::Reactor;

use Curses::UI;

open DEBUG ,'>', "/tmp/dds_watch_cui.log";

my $cui = Curses::UI->new;

my $win = $cui->add('window_id','Window');

my $label = $win->add(
	'label','Label',
	-width => 100,
	-height => 30,
	-paddingspaces => 1,
	-text => '...',
	-y => 1,
);

$label->draw;

my $bus = Net::DBus->system();

my $obj_path = "/com/deepin/download/service";
my $interfaces = "com.deepin.download.service";

my $dds = $bus->get_service("com.deepin.download.service");

my $object = $dds->get_object($obj_path, $interfaces );


my %task_info;

sub update_task_info {
	my ($taskid,$status,$message) = @_;
	my ( $old_status, $old_message );
	if ( exists $task_info{ $taskid } ){
		($old_status, $old_message ) = @{ $task_info{ $taskid } };
	}
	
	if ( ! defined $status){
		$status= $old_status // "???"; 
	}

	if( ! defined $message ){
		$message = $old_message // "???";
	}

	$task_info{ $taskid } = [ $status, $message ];

	{
		select DEBUG;
		#auto flush
		local $| = 1; 
		print DEBUG "\n=> $taskid\n[$status] \n$message\n";

	}
}


sub draw_update {
	my $text;

	for my $id ( sort keys %task_info ){
		my ( $status, $message ) = @{ $task_info{ $id } };
		$text .= join "\n", $id, $message, "[$status]","\n";
	}
	
	$label->text( $text );
	print DEBUG "draw_update() : before draw\n";
	$label->draw;
	print DEBUG "draw_update() : after draw\n";
}

sub signal_wait_handler {
	my $taskid = shift;
	update_task_info( $taskid, 'wait' );
	draw_update
}

sub signal_start_handler {
	my $taskid = shift;
	update_task_info( $taskid, 'start' );
	draw_update
}

sub signal_update_handler {
	my ($taskid, $progress, $speed, $finish,$total,$downloadSize, $totalSize ) = @_;
	$speed = sprintf "%.1f", $speed / 1024;
	$downloadSize = int( $downloadSize / 1048576);
	$totalSize = int($totalSize/ 1048576);

	update_task_info($taskid, undef ,"$progress% $speed KB/s ($finish/$total) [$downloadSize/$totalSize]");
	draw_update

}

sub signal_finish_handler {
	my $taskid = shift;
	update_task_info( $taskid, 'finish' );
	draw_update
}


sub signal_stop_handler {
	my $taskid = shift;
	update_task_info( $taskid, 'stop' );
	draw_update
}

sub signal_pause_handler {
	my $taskid = shift;
	update_task_info( $taskid, 'pause' );
	draw_update
}

sub signal_resume_handler {
	my $taskid = shift;
	update_task_info( $taskid, 'resume' );
	draw_update
}

sub signal_error_handler {
	my $taskid = shift;
	update_task_info( $taskid, 'error' );
	draw_update
}
$object->connect_to_signal( "Wait", \&signal_wait_handler );
$object->connect_to_signal( "Start", \&signal_start_handler );
$object->connect_to_signal( "Update", \&signal_update_handler );
$object->connect_to_signal( "Finish", \&signal_finish_handler );
$object->connect_to_signal( "Stop", \&signal_stop_handler );
$object->connect_to_signal( "Pause", \&signal_pause_handler );
$object->connect_to_signal( "Resume", \&signal_resume_handler );
$object->connect_to_signal( "Error", \&signal_error_handler );

my $reactor = Net::DBus::Reactor->main();
$reactor->run;

END {
	close DEBUG;
}
