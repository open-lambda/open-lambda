# Copyright © 1998 Richard Braakman
# Copyright © 1999 Darren Benham
# Copyright © 2000 Sean 'Shaleh' Perry
# Copyright © 2004 Frank Lichtenheld
# Copyright © 2006 Russ Allbery
# Copyright © 2007-2009 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2008-2009, 2012-2014 Guillem Jover <guillem@debian.org>
#
# This program is free software; you may redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation; either version 2 of the License, or
# (at your option) any later version.
#
# This is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

package Dpkg::Deps::Multiple;

=encoding utf8

=head1 NAME

Dpkg::Deps::Multiple - base module to represent multiple dependencies

=head1 DESCRIPTION

The Dpkg::Deps::Multiple module provides objects implementing various types
of dependencies. It is the base class for Dpkg::Deps::{AND,OR,Union}.

=cut

use strict;
use warnings;

our $VERSION = '1.02';

use Carp;

use Dpkg::ErrorHandling;

use parent qw(Dpkg::Interface::Storable);

=head1 METHODS

=over 4

=item $dep = Dpkg::Deps::Multiple->new(%opts);

Creates a new object.

=cut

sub new {
    my $this = shift;
    my $class = ref($this) || $this;
    my $self = { list => [ @_ ] };

    bless $self, $class;
    return $self;
}

=item $dep->reset()

Clears any dependency information stored in $dep so that $dep->is_empty()
returns true.

=cut

sub reset {
    my $self = shift;

    $self->{list} = [];
}

=item $dep->add(@deps)

Adds new dependency objects at the end of the list.

=cut

sub add {
    my $self = shift;

    push @{$self->{list}}, @_;
}

=item $dep->get_deps()

Returns a list of sub-dependencies.

=cut

sub get_deps {
    my $self = shift;

    return grep { not $_->is_empty() } @{$self->{list}};
}

=item $dep->sort()

Sorts alphabetically the internal list of dependencies.

=cut

sub sort {
    my $self = shift;

    my @res = ();
    @res = sort { Dpkg::Deps::deps_compare($a, $b) } @{$self->{list}};
    $self->{list} = [ @res ];
}

=item $dep->arch_is_concerned($arch)

Returns true if at least one of the sub-dependencies apply to this
architecture.

=cut

sub arch_is_concerned {
    my ($self, $host_arch) = @_;

    my $res = 0;
    foreach my $dep (@{$self->{list}}) {
        $res = 1 if $dep->arch_is_concerned($host_arch);
    }
    return $res;
}

=item $dep->reduce_arch($arch)

Simplifies the dependencies to contain only information relevant to the
given architecture. The non-relevant sub-dependencies are simply removed.

This trims off the architecture restriction list of Dpkg::Deps::Simple
objects.

=cut

sub reduce_arch {
    my ($self, $host_arch) = @_;

    my @new;
    foreach my $dep (@{$self->{list}}) {
        $dep->reduce_arch($host_arch);
        push @new, $dep if $dep->arch_is_concerned($host_arch);
    }
    $self->{list} = [ @new ];
}

=item $dep->has_arch_restriction()

Returns the list of package names that have such a restriction.

=cut

sub has_arch_restriction {
    my $self = shift;

    my @res;
    foreach my $dep (@{$self->{list}}) {
        push @res, $dep->has_arch_restriction();
    }
    return @res;
}

=item $dep->profile_is_concerned()

Returns true if at least one of the sub-dependencies apply to this profile.

=cut

sub profile_is_concerned {
    my ($self, $build_profiles) = @_;

    my $res = 0;
    foreach my $dep (@{$self->{list}}) {
        $res = 1 if $dep->profile_is_concerned($build_profiles);
    }
    return $res;
}

=item $dep->reduce_profiles()

Simplifies the dependencies to contain only information relevant to the
given profile. The non-relevant sub-dependencies are simply removed.

This trims off the profile restriction list of Dpkg::Deps::Simple objects.

=cut

sub reduce_profiles {
    my ($self, $build_profiles) = @_;

    my @new;
    foreach my $dep (@{$self->{list}}) {
        $dep->reduce_profiles($build_profiles);
        push @new, $dep if $dep->profile_is_concerned($build_profiles);
    }
    $self->{list} = [ @new ];
}

=item $dep->is_empty()

Returns true if the dependency is empty and doesn't contain any useful
information. This is true when a (descendant of) Dpkg::Deps::Multiple
contains an empty list of dependencies.

=cut

sub is_empty {
    my $self = shift;

    return scalar @{$self->{list}} == 0;
}

=item $dep->merge_union($other_dep)

This method is not meaningful for this object, and will always croak.

=cut

sub merge_union {
    croak 'method merge_union() is only valid for Dpkg::Deps::Simple';
}

=back

=head1 CHANGES

=head2 Version 1.02 (dpkg 1.17.10)

New methods: Add $dep->profile_is_concerned() and $dep->reduce_profiles().

=head2 Version 1.01 (dpkg 1.16.1)

New method: Add $dep->reset().

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
