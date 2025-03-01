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

package Dpkg::Control::Tests;

use strict;
use warnings;

our $VERSION = '1.00';

use Dpkg::Control;
use Dpkg::Control::Tests::Entry;
use Dpkg::Index;

use parent qw(Dpkg::Index);

=encoding utf8

=head1 NAME

Dpkg::Control::Tests - parse files like debian/tests/control

=head1 DESCRIPTION

It provides a class to access data of files that follow the same
syntax as F<debian/tests/control>.

=head1 METHODS

All the methods of Dpkg::Index are available. Those listed below are either
new or overridden with a different behavior.

=over 4

=item $c = Dpkg::Control::Tests->new(%opts)

Create a new Dpkg::Control::Tests object, which inherits from Dpkg::Index.

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;
    my $self = Dpkg::Index->new(type => CTRL_TESTS, %opts);

    return bless $self, $class;
}

=item $item = $tests->new_item()

Creates a new item.

=cut

sub new_item {
    my $self = shift;

    return Dpkg::Control::Tests::Entry->new();
}

=back

=head1 CHANGES

=head2 Version 1.00 (dpkg 1.18.8)

Mark the module as public.

=cut

1;
