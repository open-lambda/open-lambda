# Copyright © 2002 Adam Heath <doogie@debian.org>
# Copyright © 2012-2013 Guillem Jover <guillem@debian.org>
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

package Dpkg::Exit;

use strict;
use warnings;

our $VERSION = '2.00';
our @EXPORT_OK = qw(
    push_exit_handler
    pop_exit_handler
    run_exit_handlers
);

use Exporter qw(import);

my @handlers = ();

=encoding utf8

=head1 NAME

Dpkg::Exit - program exit handlers

=head1 DESCRIPTION

The Dpkg::Exit module provides support functions to run handlers on exit.

=head1 FUNCTIONS

=over 4

=item push_exit_handler($func)

Register a code reference into the exit function handlers stack.

=cut

sub push_exit_handler {
    my ($func) = shift;

    _setup_exit_handlers() if @handlers == 0;
    push @handlers, $func;
}

=item pop_exit_handler()

Pop the last registered exit handler from the handlers stack.

=cut

sub pop_exit_handler {
    _reset_exit_handlers() if @handlers == 1;
    pop @handlers;
}

=item run_exit_handlers()

Run the registered exit handlers.

=cut

sub run_exit_handlers {
    while (my $handler = pop @handlers) {
        $handler->();
    }
    _reset_exit_handlers();
}

sub _exit_handler {
    run_exit_handlers();
    exit(127);
}

my @SIGNAMES = qw(INT HUP QUIT);
my %SIGOLD;

sub _setup_exit_handlers
{
    foreach my $signame (@SIGNAMES) {
        $SIGOLD{$signame} = $SIG{$signame};
        $SIG{$signame} = \&_exit_handler;
    }
}

sub _reset_exit_handlers
{
    foreach my $signame (@SIGNAMES) {
        $SIG{$signame} = $SIGOLD{$signame};
    }
}

END {
    local $?;
    run_exit_handlers();
}

=back

=head1 CHANGES

=head2 Version 2.00 (dpkg 1.20.0)

Hide variable: @handlers.

=head2 Version 1.01 (dpkg 1.17.2)

New functions: push_exit_handler(), pop_exit_handler(), run_exit_handlers()

Deprecated variable: @handlers

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
