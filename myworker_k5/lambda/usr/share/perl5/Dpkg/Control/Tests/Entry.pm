# Copyright © 2016 Guillem Jover <guillem@debian.org>
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

package Dpkg::Control::Tests::Entry;

use strict;
use warnings;

our $VERSION = '1.00';

use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Control;

use parent qw(Dpkg::Control);

=encoding utf8

=head1 NAME

Dpkg::Control::Tests::Entry - represents a test suite entry

=head1 DESCRIPTION

This class represents a test suite entry.

=head1 METHODS

All the methods of Dpkg::Control are available. Those listed below are either
new or overridden with a different behavior.

=over 4

=item $entry = Dpkg::Control::Tests::Entry->new()

Creates a new object. It does not represent a real control test entry
until one has been successfully parsed or built from scratch.

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;

    my $self = Dpkg::Control->new(type => CTRL_TESTS, %opts);
    bless $self, $class;
    return $self;
}

=item $entry->parse($fh, $desc)

Parse a control test entry from a filehandle. When called multiple times,
the parsed fields are accumulated.

Returns true if parsing was a success.

=cut

sub parse {
    my ($self, $fh, $desc) = @_;

    return if not $self->SUPER::parse($fh, $desc);

    if (not exists $self->{'Tests'} and not exists $self->{'Test-Command'}) {
        $self->parse_error($desc, g_('block lacks either %s or %s fields'),
                           'Tests', 'Test-Command');
    }

    return 1;
}

=back

=head1 CHANGES

=head2 Version 1.00 (dpkg 1.18.8)

Mark the module as public.

=cut

1;
