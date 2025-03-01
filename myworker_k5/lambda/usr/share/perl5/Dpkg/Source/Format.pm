# Copyright © 2008-2011 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2008-2018 Guillem Jover <guillem@debian.org>
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

package Dpkg::Source::Format;

=encoding utf8

=head1 NAME

Dpkg::Source::Format - manipulate debian/source/format files

=head1 DESCRIPTION

This module provides a class that can manipulate Debian source
package F<debian/source/format> files.

=cut

use strict;
use warnings;

our $VERSION = '1.00';

use Dpkg::Gettext;
use Dpkg::ErrorHandling;

use parent qw(Dpkg::Interface::Storable);

=head1 METHODS

=over 4

=item $f = Dpkg::Source::Format->new(%opts)

Creates a new object corresponding to a source package's
F<debian/source/format> file. When the key B<filename> is set, it will
be used to parse and set the format. Otherwise if the B<format> key is
set it will be validated and used to set the format.

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;
    my $self = {
        filename => undef,
        major => undef,
        minor => undef,
        variant => undef,
    };
    bless $self, $class;

    if (exists $opts{filename}) {
        $self->load($opts{filename}, compression => 0);
    } elsif ($opts{format}) {
        $self->set($opts{format});
    }
    return $self;
}

=item $f->set_from_parts($major[, $minor[, $variant]])

Sets the source format from its parts. The $major part is mandatory.
The $minor and $variant parts are optional.

B<Notice>: This function performs no validation.

=cut

sub set_from_parts {
    my ($self, $major, $minor, $variant) = @_;

    $self->{major} = $major;
    $self->{minor} = $minor // 0;
    $self->{variant} = $variant;
}

=item ($major, $minor, $variant) = $f->set($format)

Sets (and validates) the source $format specified. Will return the parsed
format parts as a list, the optional $minor and $variant parts might be
undef.

=cut

sub set {
    my ($self, $format) = @_;

    if ($format =~ /^(\d+)(?:\.(\d+))?(?:\s+\(([a-z0-9]+)\))?$/) {
        my ($major, $minor, $variant) = ($1, $2, $3);

        $self->set_from_parts($major, $minor, $variant);

        return ($major, $minor, $variant);
    } else {
        error(g_("source package format '%s' is invalid"), $format);
    }
}

=item ($major, $minor, $variant) = $f->get()

=item $format = $f->get()

Gets the source format, either as properly formatted scalar, or as a list
of its parts, where the optional $minor and $variant parts might be undef.

=cut

sub get {
    my $self = shift;

    if (wantarray) {
        return ($self->{major}, $self->{minor}, $self->{variant});
    } else {
        my $format = "$self->{major}.$self->{minor}";
        $format .= " ($self->{variant})" if defined $self->{variant};

        return $format;
    }
}

=item $count = $f->parse($fh, $desc)

Parse the source format string from $fh, with filehandle description $desc.

=cut

sub parse {
    my ($self, $fh, $desc) = @_;

    my $format = <$fh>;
    chomp $format if defined $format;
    error(g_('%s is empty'), $desc)
        unless defined $format and length $format;

    $self->set($format);

    return 1;
}

=item $count = $f->load($filename)

Parse $filename contents for a source package format string.

=item $str = $f->output([$fh])

=item "$f"

Returns a string representing the source package format version.
If $fh is set, it prints the string to the filehandle.

=cut

sub output {
    my ($self, $fh) = @_;

    my $str = $self->get();

    print { $fh } "$str\n" if defined $fh;

    return $str;
}

=item $f->save($filename)

Save the source package format into the given $filename.

=back

=head1 CHANGES

=head2 Version 1.00 (dpkg 1.19.3)

Mark the module as public.

=cut

1;
