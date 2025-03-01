# Copyright © 2009 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::Changelog::Entry;

use strict;
use warnings;

our $VERSION = '1.01';

use Carp;

use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Control::Changelog;

use overload
    '""' => \&output,
    'eq' => sub { defined($_[1]) and "$_[0]" eq "$_[1]" },
    fallback => 1;

=encoding utf8

=head1 NAME

Dpkg::Changelog::Entry - represents a changelog entry

=head1 DESCRIPTION

This class represents a changelog entry. It is composed
of a set of lines with specific purpose: a header line, changes lines, a
trailer line. Blank lines can be between those kind of lines.

=head1 METHODS

=over 4

=item $entry = Dpkg::Changelog::Entry->new()

Creates a new object. It doesn't represent a real changelog entry
until one has been successfully parsed or built from scratch.

=cut

sub new {
    my $this = shift;
    my $class = ref($this) || $this;

    my $self = {
	header => undef,
	changes => [],
	trailer => undef,
	blank_after_header => [],
	blank_after_changes => [],
	blank_after_trailer => [],
    };
    bless $self, $class;
    return $self;
}

=item $str = $entry->output()

=item "$entry"

Get a string representation of the changelog entry.

=item $entry->output($fh)

Print the string representation of the changelog entry to a
filehandle.

=cut

sub _format_output_block {
    my $lines = shift;
    return join('', map { $_ . "\n" } @{$lines});
}

sub output {
    my ($self, $fh) = @_;
    my $str = '';
    $str .= $self->{header} . "\n" if defined($self->{header});
    $str .= _format_output_block($self->{blank_after_header});
    $str .= _format_output_block($self->{changes});
    $str .= _format_output_block($self->{blank_after_changes});
    $str .= $self->{trailer} . "\n" if defined($self->{trailer});
    $str .= _format_output_block($self->{blank_after_trailer});
    print { $fh } $str if defined $fh;
    return $str;
}

=item $entry->get_part($part)

Return either a string (for a single line) or an array ref (for multiple
lines) corresponding to the requested part. $part can be
"header, "changes", "trailer", "blank_after_header",
"blank_after_changes", "blank_after_trailer".

=cut

sub get_part {
    my ($self, $part) = @_;
    croak "invalid part of changelog entry: $part" unless exists $self->{$part};
    return $self->{$part};
}

=item $entry->set_part($part, $value)

Set the value of the corresponding part. $value can be a string
or an array ref.

=cut

sub set_part {
    my ($self, $part, $value) = @_;
    croak "invalid part of changelog entry: $part" unless exists $self->{$part};
    if (ref($self->{$part})) {
	if (ref($value)) {
	    $self->{$part} = $value;
	} else {
	    $self->{$part} = [ $value ];
	}
    } else {
	$self->{$part} = $value;
    }
}

=item $entry->extend_part($part, $value)

Concatenate $value at the end of the part. If the part is already a
multi-line value, $value is added as a new line otherwise it's
concatenated at the end of the current line.

=cut

sub extend_part {
    my ($self, $part, $value, @rest) = @_;
    croak "invalid part of changelog entry: $part" unless exists $self->{$part};
    if (ref($self->{$part})) {
	if (ref($value)) {
	    push @{$self->{$part}}, @$value;
	} else {
	    push @{$self->{$part}}, $value;
	}
    } else {
	if (defined($self->{$part})) {
	    if (ref($value)) {
		$self->{$part} = [ $self->{$part}, @$value ];
	    } else {
		$self->{$part} .= $value;
	    }
	} else {
	    $self->{$part} = $value;
	}
    }
}

=item $is_empty = $entry->is_empty()

Returns 1 if the changelog entry doesn't contain anything at all.
Returns 0 as soon as it contains something in any of its non-blank
parts.

=cut

sub is_empty {
    my $self = shift;
    return !(defined($self->{header}) || defined($self->{trailer}) ||
	     scalar(@{$self->{changes}}));
}

=item $entry->normalize()

Normalize the content. Strip whitespaces at end of lines, use a single
empty line to separate each part.

=cut

sub normalize {
    my $self = shift;
    if (defined($self->{header})) {
	$self->{header} =~ s/\s+$//g;
	$self->{blank_after_header} = [''];
    } else {
	$self->{blank_after_header} = [];
    }
    if (scalar(@{$self->{changes}})) {
	s/\s+$//g foreach @{$self->{changes}};
	$self->{blank_after_changes} = [''];
    } else {
	$self->{blank_after_changes} = [];
    }
    if (defined($self->{trailer})) {
	$self->{trailer} =~ s/\s+$//g;
	$self->{blank_after_trailer} = [''];
    } else {
	$self->{blank_after_trailer} = [];
    }
}

=item $src = $entry->get_source()

Return the name of the source package associated to the changelog entry.

=cut

sub get_source {
    return;
}

=item $ver = $entry->get_version()

Return the version associated to the changelog entry.

=cut

sub get_version {
    return;
}

=item @dists = $entry->get_distributions()

Return a list of target distributions for this version.

=cut

sub get_distributions {
    return;
}

=item $fields = $entry->get_optional_fields()

Return a set of optional fields exposed by the changelog entry.
It always returns a Dpkg::Control object (possibly empty though).

=cut

sub get_optional_fields {
    return Dpkg::Control::Changelog->new();
}

=item $urgency = $entry->get_urgency()

Return the urgency of the associated upload.

=cut

sub get_urgency {
    return;
}

=item $maint = $entry->get_maintainer()

Return the string identifying the person who signed this changelog entry.

=cut

sub get_maintainer {
    return;
}

=item $time = $entry->get_timestamp()

Return the timestamp of the changelog entry.

=cut

sub get_timestamp {
    return;
}

=item $time = $entry->get_timepiece()

Return the timestamp of the changelog entry as a Time::Piece object.

This function might return undef if there was no timestamp.

=cut

sub get_timepiece {
    return;
}

=item $str = $entry->get_dpkg_changes()

Returns a string that is suitable for usage in a C<Changes> field
in the output format of C<dpkg-parsechangelog>.

=cut

sub get_dpkg_changes {
    my $self = shift;
    my $header = $self->get_part('header') // '';
    $header =~ s/\s+$//;
    return "\n$header\n\n" . join("\n", @{$self->get_part('changes')});
}

=back

=head1 CHANGES

=head2 Version 1.01 (dpkg 1.18.8)

New method: $entry->get_timepiece().

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
