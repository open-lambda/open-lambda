# Copyright © 2008-2010 Raphaël Hertzog <hertzog@debian.org>
#
# This program is free software; you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation; either version 2 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

package Dpkg::Compression::Process;

use strict;
use warnings;

our $VERSION = '1.00';

use Carp;

use Dpkg::Compression;
use Dpkg::ErrorHandling;
use Dpkg::Gettext;
use Dpkg::IPC;

=encoding utf8

=head1 NAME

Dpkg::Compression::Process - run compression/decompression processes

=head1 DESCRIPTION

This module provides an object oriented interface to run and manage
compression/decompression processes.

=head1 METHODS

=over 4

=item $proc = Dpkg::Compression::Process->new(%opts)

Create a new instance of the object. Supported options are "compression"
and "compression_level" (see corresponding set_* functions).

=cut

sub new {
    my ($this, %args) = @_;
    my $class = ref($this) || $this;
    my $self = {};
    bless $self, $class;
    $self->set_compression($args{compression} || compression_get_default());
    $self->set_compression_level($args{compression_level} ||
	    compression_get_default_level());
    return $self;
}

=item $proc->set_compression($comp)

Select the compression method to use. It errors out if the method is not
supported according to C<compression_is_supported> (of
B<Dpkg::Compression>).

=cut

sub set_compression {
    my ($self, $method) = @_;
    error(g_('%s is not a supported compression method'), $method)
	    unless compression_is_supported($method);
    $self->{compression} = $method;
}

=item $proc->set_compression_level($level)

Select the compression level to use. It errors out if the level is not
valid according to C<compression_is_valid_level> (of
B<Dpkg::Compression>).

=cut

sub set_compression_level {
    my ($self, $level) = @_;
    error(g_('%s is not a compression level'), $level)
	    unless compression_is_valid_level($level);
    $self->{compression_level} = $level;
}

=item @exec = $proc->get_compress_cmdline()

=item @exec = $proc->get_uncompress_cmdline()

Returns a list ready to be passed to C<exec>, its first element is the
program name (either for compression or decompression) and the following
elements are parameters for the program.

When executed the program acts as a filter between its standard input
and its standard output.

=cut

sub get_compress_cmdline {
    my $self = shift;
    my @prog = (@{compression_get_property($self->{compression}, 'comp_prog')});
    my $level = '-' . $self->{compression_level};
    $level = '--' . $self->{compression_level}
	    if $self->{compression_level} !~ m/^[1-9]$/;
    push @prog, $level;
    return @prog;
}

sub get_uncompress_cmdline {
    my $self = shift;
    return (@{compression_get_property($self->{compression}, 'decomp_prog')});
}

sub _sanity_check {
    my ($self, %opts) = @_;
    # Check for proper cleaning before new start
    error(g_('Dpkg::Compression::Process can only start one subprocess at a time'))
	    if $self->{pid};
    # Check options
    my $to = my $from = 0;
    foreach my $thing (qw(file handle string pipe)) {
        $to++ if $opts{"to_$thing"};
        $from++ if $opts{"from_$thing"};
    }
    croak 'exactly one to_* parameter is needed' if $to != 1;
    croak 'exactly one from_* parameter is needed' if $from != 1;
    return %opts;
}

=item $proc->compress(%opts)

Starts a compressor program. You must indicate where it will read its
uncompressed data from and where it will write its compressed data to.
This is accomplished by passing one parameter C<to_*> and one parameter
C<from_*> as accepted by B<Dpkg::IPC::spawn>.

You must call C<wait_end_process> after having called this method to
properly close the sub-process (and verify that it exited without error).

=cut

sub compress {
    my ($self, %opts) = @_;

    $self->_sanity_check(%opts);
    my @prog = $self->get_compress_cmdline();
    $opts{exec} = \@prog;
    $self->{cmdline} = "@prog";
    $self->{pid} = spawn(%opts);
    delete $self->{pid} if $opts{to_string}; # wait_child already done
}

=item $proc->uncompress(%opts)

Starts a decompressor program. You must indicate where it will read its
compressed data from and where it will write its uncompressed data to.
This is accomplished by passing one parameter C<to_*> and one parameter
C<from_*> as accepted by B<Dpkg::IPC::spawn>.

You must call C<wait_end_process> after having called this method to
properly close the sub-process (and verify that it exited without error).

=cut

sub uncompress {
    my ($self, %opts) = @_;

    $self->_sanity_check(%opts);
    my @prog = $self->get_uncompress_cmdline();
    $opts{exec} = \@prog;
    $self->{cmdline} = "@prog";
    $self->{pid} = spawn(%opts);
    delete $self->{pid} if $opts{to_string}; # wait_child already done
}

=item $proc->wait_end_process(%opts)

Call B<Dpkg::IPC::wait_child> to wait until the sub-process has exited
and verify its return code. Any given option will be forwarded to
the C<wait_child> function. Most notably you can use the "nocheck" option
to verify the return code yourself instead of letting C<wait_child> do
it for you.

=cut

sub wait_end_process {
    my ($self, %opts) = @_;
    $opts{cmdline} //= $self->{cmdline};
    wait_child($self->{pid}, %opts) if $self->{pid};
    delete $self->{pid};
    delete $self->{cmdline};
}

=back

=head1 CHANGES

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
