# Copyright © 2009 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2012-2017 Guillem Jover <guillem@debian.org>
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

package Dpkg::Index;

use strict;
use warnings;

our $VERSION = '2.01';

use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Control;

use parent qw(Dpkg::Interface::Storable);

use overload
    '@{}' => sub { return $_[0]->{order} },
    fallback => 1;

=encoding utf8

=head1 NAME

Dpkg::Index - generic index of control information

=head1 DESCRIPTION

This class represent a set of Dpkg::Control objects.

=head1 METHODS

=over 4

=item $index = Dpkg::Index->new(%opts)

Creates a new empty index. See set_options() for more details.

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;

    my $self = {
	items => {},
	order => [],
	unique_tuple_key => 1,
	get_key_func => sub { return $_[0]->{Package} },
	type => CTRL_UNKNOWN,
        item_opts => {},
    };
    bless $self, $class;
    $self->set_options(%opts);
    if (exists $opts{load}) {
	$self->load($opts{load});
    }

    return $self;
}

=item $index->set_options(%opts)

The "type" option is checked first to define default values for other
options. Here are the relevant options: "get_key_func" is a function
returning a key for the item passed in parameters, "unique_tuple_key" is
a boolean requesting whether the default key should be the unique tuple
(default to true), "item_opts" is a hash reference that will be passed to
the item constructor in the new_item() method.
The index can only contain one item with a given key.
The "get_key_func" function used depends on the type:

=over

=item *

for CTRL_INFO_SRC, it is the Source field;

=item *

for CTRL_INDEX_SRC and CTRL_PKG_SRC it is the Package and Version fields
(concatenated with "_") when "unique_tuple_key" is true (the default), or
otherwise the Package field;

=item *

for CTRL_INFO_PKG it is simply the Package field;

=item *

for CTRL_INDEX_PKG and CTRL_PKG_DEB it is the Package, Version and
Architecture fields (concatenated with "_") when "unique_tuple_key" is
true (the default) or otherwise the Package field;

=item *

for CTRL_CHANGELOG it is the Source and the Version fields (concatenated
with an intermediary "_");

=item *

for CTRL_TESTS is either the Tests or Test-Command fields;

=item *

for CTRL_FILE_CHANGES it is the Source, Version and Architecture fields
(concatenated with "_");

=item *

for CTRL_FILE_VENDOR it is the Vendor field;

=item *

for CTRL_FILE_STATUS it is the Package and Architecture fields (concatenated
with "_");

=item *

otherwise it is the Package field by default.

=back

=cut

sub set_options {
    my ($self, %opts) = @_;

    # Default values based on type
    if (exists $opts{type}) {
        my $t = $opts{type};
        if ($t == CTRL_INFO_PKG) {
	    $self->{get_key_func} = sub { return $_[0]->{Package}; };
        } elsif ($t == CTRL_INFO_SRC) {
	    $self->{get_key_func} = sub { return $_[0]->{Source}; };
        } elsif ($t == CTRL_CHANGELOG) {
	    $self->{get_key_func} = sub {
		return $_[0]->{Source} . '_' . $_[0]->{Version};
	    };
        } elsif ($t == CTRL_COPYRIGHT_HEADER) {
            # This is a bit pointless, because the value will almost always
            # be the same, but guarantees that we use a known field.
            $self->{get_key_func} = sub { return $_[0]->{Format}; };
        } elsif ($t == CTRL_COPYRIGHT_FILES) {
            $self->{get_key_func} = sub { return $_[0]->{Files}; };
        } elsif ($t == CTRL_COPYRIGHT_LICENSE) {
            $self->{get_key_func} = sub { return $_[0]->{License}; };
        } elsif ($t == CTRL_TESTS) {
            $self->{get_key_func} = sub {
                return $_[0]->{Tests} || $_[0]->{'Test-Command'};
            };
        } elsif ($t == CTRL_INDEX_SRC or $t == CTRL_PKG_SRC) {
            if ($opts{unique_tuple_key} // $self->{unique_tuple_key}) {
                $self->{get_key_func} = sub {
                    return $_[0]->{Package} . '_' . $_[0]->{Version};
                };
            } else {
                $self->{get_key_func} = sub {
                    return $_[0]->{Package};
                };
            }
        } elsif ($t == CTRL_INDEX_PKG or $t == CTRL_PKG_DEB) {
            if ($opts{unique_tuple_key} // $self->{unique_tuple_key}) {
                $self->{get_key_func} = sub {
                    return $_[0]->{Package} . '_' . $_[0]->{Version} . '_' .
                           $_[0]->{Architecture};
                };
            } else {
                $self->{get_key_func} = sub {
                    return $_[0]->{Package};
                };
            }
        } elsif ($t == CTRL_FILE_CHANGES) {
	    $self->{get_key_func} = sub {
		return $_[0]->{Source} . '_' . $_[0]->{Version} . '_' .
		       $_[0]->{Architecture};
	    };
        } elsif ($t == CTRL_FILE_VENDOR) {
	    $self->{get_key_func} = sub { return $_[0]->{Vendor}; };
        } elsif ($t == CTRL_FILE_STATUS) {
	    $self->{get_key_func} = sub {
		return $_[0]->{Package} . '_' . $_[0]->{Architecture};
	    };
        }
    }

    # Options set by the user override default values
    $self->{$_} = $opts{$_} foreach keys %opts;
}

=item $index->get_type()

Returns the type of control information stored. See the type parameter
set during new().

=cut

sub get_type {
    my $self = shift;
    return $self->{type};
}

=item $index->add($item, [$key])

Add a new item in the index. If the $key parameter is omitted, the key
will be generated with the get_key_func function (see set_options() for
details).

=cut

sub add {
    my ($self, $item, $key) = @_;

    $key //= $self->{get_key_func}($item);
    if (not exists $self->{items}{$key}) {
	push @{$self->{order}}, $key;
    }
    $self->{items}{$key} = $item;
}

=item $index->parse($fh, $desc)

Reads the filehandle and creates all items parsed. When called multiple
times, the parsed stanzas are accumulated.

Returns the number of items parsed.

=cut

sub parse {
    my ($self, $fh, $desc) = @_;
    my $item = $self->new_item();
    my $i = 0;
    while ($item->parse($fh, $desc)) {
	$self->add($item);
	$item = $self->new_item();
	$i++;
    }
    return $i;
}

=item $index->load($file)

Reads the file and creates all items parsed. Returns the number of items
parsed. Handles compressed files transparently based on their extensions.

=item $item = $index->new_item()

Creates a new item. Mainly useful for derived objects that would want
to override this method to return something else than a Dpkg::Control
object.

=cut

sub new_item {
    my $self = shift;
    return Dpkg::Control->new(%{$self->{item_opts}}, type => $self->{type});
}

=item $item = $index->get_by_key($key)

Returns the item identified by $key or undef.

=cut

sub get_by_key {
    my ($self, $key) = @_;
    return $self->{items}{$key} if exists $self->{items}{$key};
    return;
}

=item @keys = $index->get_keys(%criteria)

Returns the keys of items that matches all the criteria. The key of the
%criteria hash is a field name and the value is either a regex that needs
to match the field value, or a reference to a function that must return
true and that receives the field value as single parameter, or a scalar
that must be equal to the field value.

=cut

sub get_keys {
    my ($self, %crit) = @_;
    my @selected = @{$self->{order}};
    foreach my $s_crit (keys %crit) { # search criteria
	if (ref($crit{$s_crit}) eq 'Regexp') {
	    @selected = grep {
		exists $self->{items}{$_}{$s_crit} and
		       $self->{items}{$_}{$s_crit} =~ $crit{$s_crit}
	    } @selected;
	} elsif (ref($crit{$s_crit}) eq 'CODE') {
	    @selected = grep {
		$crit{$s_crit}->($self->{items}{$_}{$s_crit});
	    } @selected;
	} else {
	    @selected = grep {
		exists $self->{items}{$_}{$s_crit} and
		       $self->{items}{$_}{$s_crit} eq $crit{$s_crit}
	    } @selected;
	}
    }
    return @selected;
}

=item @items = $index->get(%criteria)

Returns all the items that matches all the criteria.

=cut

sub get {
    my ($self, %crit) = @_;
    return map { $self->{items}{$_} } $self->get_keys(%crit);
}

=item $index->remove_by_key($key)

Remove the item identified by the given key.

=cut

sub remove_by_key {
    my ($self, $key) = @_;
    @{$self->{order}} = grep { $_ ne $key } @{$self->{order}};
    return delete $self->{items}{$key};
}

=item @items = $index->remove(%criteria)

Returns and removes all the items that matches all the criteria.

=cut

sub remove {
    my ($self, %crit) = @_;
    my @keys = $self->get_keys(%crit);
    my (%keys, @ret);
    foreach my $key (@keys) {
	$keys{$key} = 1;
	push @ret, $self->{items}{$key} if defined wantarray;
	delete $self->{items}{$key};
    }
    @{$self->{order}} = grep { not exists $keys{$_} } @{$self->{order}};
    return @ret;
}

=item $index->merge($other_index, %opts)

Merge the entries of the other index. While merging, the keys of the merged
index are used, they are not re-computed (unless you have set the options
"keep_keys" to "0"). It's your responsibility to ensure that they have been
computed with the same function.

=cut

sub merge {
    my ($self, $other, %opts) = @_;
    $opts{keep_keys} //= 1;
    foreach my $key ($other->get_keys()) {
	$self->add($other->get_by_key($key), $opts{keep_keys} ? $key : undef);
    }
}

=item $index->sort(\&sortfunc)

Sort the index with the given sort function. If no function is given, an
alphabetic sort is done based on the keys. The sort function receives the
items themselves as parameters and not the keys.

=cut

sub sort {
    my ($self, $func) = @_;
    if (defined $func) {
	@{$self->{order}} = sort {
	    $func->($self->{items}{$a}, $self->{items}{$b})
	} @{$self->{order}};
    } else {
	@{$self->{order}} = sort @{$self->{order}};
    }
}

=item $str = $index->output([$fh])

=item "$index"

Get a string representation of the index. The L<Dpkg::Control> objects are
output in the order which they have been read or added except if the order
have been changed with sort().

Print the string representation of the index to a filehandle if $fh has
been passed.

=cut

sub output {
    my ($self, $fh) = @_;
    my $str = '';
    foreach my $key ($self->get_keys()) {
	if (defined $fh) {
	    print { $fh } $self->get_by_key($key) . "\n";
	}
	if (defined wantarray) {
	    $str .= $self->get_by_key($key) . "\n";
	}
    }
    return $str;
}

=item $index->save($file)

Writes the content of the index in a file. Auto-compresses files
based on their extensions.

=back

=head1 CHANGES

=head2 Version 2.01 (dpkg 1.20.6)

New option: Add new "item_opts" option.

=head2 Version 2.00 (dpkg 1.20.0)

Change behavior: The "unique_tuple_key" option now defaults to true.

=head2 Version 1.01 (dpkg 1.19.0)

New option: Add new "unique_tuple_key" option to $index->set_options() to set
better default "get_key_func" options, which will become the default behavior
in 1.20.x.

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
