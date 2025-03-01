# Copyright © 2008 Frank Lichtenheld <djpig@debian.org>
# Copyright © 2008, 2012-2015 Guillem Jover <guillem@debian.org>
# Copyright © 2010 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::Checksums;

use strict;
use warnings;

our $VERSION = '1.04';
our @EXPORT = qw(
    checksums_is_supported
    checksums_get_list
    checksums_get_property
);

use Exporter qw(import);
use Digest;

use Dpkg::Gettext;
use Dpkg::ErrorHandling;

=encoding utf8

=head1 NAME

Dpkg::Checksums - generate and manipulate file checksums

=head1 DESCRIPTION

This module provides a class that can generate and manipulate
various file checksums as well as some methods to query information
about supported checksums.

=head1 FUNCTIONS

=over 4

=cut

my $CHECKSUMS = {
    md5 => {
	name => 'MD5',
	regex => qr/[0-9a-f]{32}/,
	strong => 0,
    },
    sha1 => {
	name => 'SHA-1',
	regex => qr/[0-9a-f]{40}/,
	strong => 0,
    },
    sha256 => {
	name => 'SHA-256',
	regex => qr/[0-9a-f]{64}/,
	strong => 1,
    },
};

=item @list = checksums_get_list()

Returns the list of supported checksums algorithms.

=cut

sub checksums_get_list() {
    my @list = sort keys %{$CHECKSUMS};
    return @list;
}

=item $bool = checksums_is_supported($alg)

Returns a boolean indicating whether the given checksum algorithm is
supported. The checksum algorithm is case-insensitive.

=cut

sub checksums_is_supported($) {
    my $alg = shift;
    return exists $CHECKSUMS->{lc($alg)};
}

=item $value = checksums_get_property($alg, $property)

Returns the requested property of the checksum algorithm. Returns undef if
either the property or the checksum algorithm doesn't exist. Valid
properties currently include "name" (returns the name of the digest
algorithm), "regex" for the regular expression describing the common
string representation of the checksum, and "strong" for a boolean describing
whether the checksum algorithm is considered cryptographically strong.

=cut

sub checksums_get_property($$) {
    my ($alg, $property) = @_;

    return unless checksums_is_supported($alg);
    return $CHECKSUMS->{lc($alg)}{$property};
}

=back

=head1 METHODS

=over 4

=item $ck = Dpkg::Checksums->new()

Create a new Dpkg::Checksums object. This object is able to store
the checksums of several files to later export them or verify them.

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;

    my $self = {};
    bless $self, $class;
    $self->reset();

    return $self;
}

=item $ck->reset()

Forget about all checksums stored. The object is again in the same state
as if it was newly created.

=cut

sub reset {
    my $self = shift;

    $self->{files} = [];
    $self->{checksums} = {};
    $self->{size} = {};
}

=item $ck->add_from_file($filename, %opts)

Add or verify checksums information for the file $filename. The file must
exists for the call to succeed. If you don't want the given filename to
appear when you later export the checksums you might want to set the "key"
option with the public name that you want to use. Also if you don't want
to generate all the checksums, you can pass an array reference of the
wanted checksums in the "checksums" option.

It the object already contains checksums information associated the
filename (or key), it will error out if the newly computed information
does not match what's stored, and the caller did not request that it be
updated with the boolean "update" option.

=cut

sub add_from_file {
    my ($self, $file, %opts) = @_;
    my $key = exists $opts{key} ? $opts{key} : $file;
    my @alg;
    if (exists $opts{checksums}) {
	push @alg, map { lc } @{$opts{checksums}};
    } else {
	push @alg, checksums_get_list();
    }

    push @{$self->{files}}, $key unless exists $self->{size}{$key};
    (my @s = stat($file)) or syserr(g_('cannot fstat file %s'), $file);
    if (not $opts{update} and exists $self->{size}{$key} and
        $self->{size}{$key} != $s[7]) {
	error(g_('file %s has size %u instead of expected %u'),
	      $file, $s[7], $self->{size}{$key});
    }
    $self->{size}{$key} = $s[7];

    foreach my $alg (@alg) {
        my $digest = Digest->new($CHECKSUMS->{$alg}{name});
        open my $fh, '<', $file or syserr(g_('cannot open file %s'), $file);
        $digest->addfile($fh);
        close $fh;

        my $newsum = $digest->hexdigest;
        if (not $opts{update} and exists $self->{checksums}{$key}{$alg} and
            $self->{checksums}{$key}{$alg} ne $newsum) {
            error(g_('file %s has checksum %s instead of expected %s (algorithm %s)'),
                  $file, $newsum, $self->{checksums}{$key}{$alg}, $alg);
        }
        $self->{checksums}{$key}{$alg} = $newsum;
    }
}

=item $ck->add_from_string($alg, $value, %opts)

Add checksums of type $alg that are stored in the $value variable.
$value can be multi-lines, each line should be a space separated list
of checksum, file size and filename. Leading or trailing spaces are
not allowed.

It the object already contains checksums information associated to the
filenames, it will error out if the newly read information does not match
what's stored, and the caller did not request that it be updated with
the boolean "update" option.

=cut

sub add_from_string {
    my ($self, $alg, $fieldtext, %opts) = @_;
    $alg = lc($alg);
    my $rx_fname = qr/[0-9a-zA-Z][-+:.,=0-9a-zA-Z_~]+/;
    my $regex = checksums_get_property($alg, 'regex');
    my $checksums = $self->{checksums};

    for my $checksum (split /\n */, $fieldtext) {
	next if $checksum eq '';
	unless ($checksum =~ m/^($regex)\s+(\d+)\s+($rx_fname)$/) {
	    error(g_('invalid line in %s checksums string: %s'),
		  $alg, $checksum);
	}
	my ($sum, $size, $file) = ($1, $2, $3);
	if (not $opts{update} and  exists($checksums->{$file}{$alg})
	    and $checksums->{$file}{$alg} ne $sum) {
	    error(g_("conflicting checksums '%s' and '%s' for file '%s'"),
		  $checksums->{$file}{$alg}, $sum, $file);
	}
	if (not $opts{update} and exists $self->{size}{$file}
	    and $self->{size}{$file} != $size) {
	    error(g_("conflicting file sizes '%u' and '%u' for file '%s'"),
		  $self->{size}{$file}, $size, $file);
	}
	push @{$self->{files}}, $file unless exists $self->{size}{$file};
	$checksums->{$file}{$alg} = $sum;
	$self->{size}{$file} = $size;
    }
}

=item $ck->add_from_control($control, %opts)

Read checksums from Checksums-* fields stored in the Dpkg::Control object
$control. It uses $self->add_from_string() on the field values to do the
actual work.

If the option "use_files_for_md5" evaluates to true, then the "Files"
field is used in place of the "Checksums-Md5" field. By default the option
is false.

=cut

sub add_from_control {
    my ($self, $control, %opts) = @_;
    $opts{use_files_for_md5} //= 0;
    foreach my $alg (checksums_get_list()) {
	my $key = "Checksums-$alg";
	$key = 'Files' if ($opts{use_files_for_md5} and $alg eq 'md5');
	if (exists $control->{$key}) {
	    $self->add_from_string($alg, $control->{$key}, %opts);
	}
    }
}

=item @files = $ck->get_files()

Return the list of files whose checksums are stored in the object.

=cut

sub get_files {
    my $self = shift;
    return @{$self->{files}};
}

=item $bool = $ck->has_file($file)

Return true if we have checksums for the given file. Returns false
otherwise.

=cut

sub has_file {
    my ($self, $file) = @_;
    return exists $self->{size}{$file};
}

=item $ck->remove_file($file)

Remove all checksums of the given file.

=cut

sub remove_file {
    my ($self, $file) = @_;
    return unless $self->has_file($file);
    delete $self->{checksums}{$file};
    delete $self->{size}{$file};
    @{$self->{files}} = grep { $_ ne $file } $self->get_files();
}

=item $checksum = $ck->get_checksum($file, $alg)

Return the checksum of type $alg for the requested $file. This will not
compute the checksum but only return the checksum stored in the object, if
any.

If $alg is not defined, it returns a reference to a hash: keys are
the checksum algorithms and values are the checksums themselves. The
hash returned must not be modified, it's internal to the object.

=cut

sub get_checksum {
    my ($self, $file, $alg) = @_;
    $alg = lc($alg) if defined $alg;
    if (exists $self->{checksums}{$file}) {
	return $self->{checksums}{$file} unless defined $alg;
	return $self->{checksums}{$file}{$alg};
    }
    return;
}

=item $size = $ck->get_size($file)

Return the size of the requested file if it's available in the object.

=cut

sub get_size {
    my ($self, $file) = @_;
    return $self->{size}{$file};
}

=item $bool = $ck->has_strong_checksums($file)

Return a boolean on whether the file has a strong checksum.

=cut

sub has_strong_checksums {
    my ($self, $file) = @_;

    foreach my $alg (checksums_get_list()) {
        return 1 if defined $self->get_checksum($file, $alg) and
                    checksums_get_property($alg, 'strong');
    }

    return 0;
}

=item $ck->export_to_string($alg, %opts)

Return a multi-line string containing the checksums of type $alg. The
string can be stored as-is in a Checksum-* field of a Dpkg::Control
object.

=cut

sub export_to_string {
    my ($self, $alg, %opts) = @_;
    my $res = '';
    foreach my $file ($self->get_files()) {
	my $sum = $self->get_checksum($file, $alg);
	my $size = $self->get_size($file);
	next unless defined $sum and defined $size;
	$res .= "\n$sum $size $file";
    }
    return $res;
}

=item $ck->export_to_control($control, %opts)

Export the checksums in the Checksums-* fields of the Dpkg::Control
$control object.

=cut

sub export_to_control {
    my ($self, $control, %opts) = @_;
    $opts{use_files_for_md5} //= 0;
    foreach my $alg (checksums_get_list()) {
	my $key = "Checksums-$alg";
	$key = 'Files' if ($opts{use_files_for_md5} and $alg eq 'md5');
	$control->{$key} = $self->export_to_string($alg, %opts);
    }
}

=back

=head1 CHANGES

=head2 Version 1.04 (dpkg 1.20.0)

Remove warning: For obsolete property 'program'.

=head2 Version 1.03 (dpkg 1.18.5)

New property: Add new 'strong' property.

New member: $ck->has_strong_checksums().

=head2 Version 1.02 (dpkg 1.18.0)

Obsolete property: Getting the 'program' checksum property will warn and
return undef, the Digest module is used internally now.

New property: Add new 'name' property with the name of the Digest algorithm
to use.

=head2 Version 1.01 (dpkg 1.17.6)

New argument: Accept an options argument in $ck->export_to_string().

New option: Accept new option 'update' in $ck->add_from_file() and
$ck->add_from_control().

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
