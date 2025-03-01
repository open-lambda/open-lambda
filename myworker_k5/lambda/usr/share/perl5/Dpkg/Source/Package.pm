# Copyright © 2008-2011 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2008-2019 Guillem Jover <guillem@debian.org>
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

package Dpkg::Source::Package;

=encoding utf8

=head1 NAME

Dpkg::Source::Package - manipulate Debian source packages

=head1 DESCRIPTION

This module provides a class that can manipulate Debian source
packages. While it supports both the extraction and the creation
of source packages, the only API that is officially supported
is the one that supports the extraction of the source package.

=cut

use strict;
use warnings;

our $VERSION = '2.01';
our @EXPORT_OK = qw(
    get_default_diff_ignore_regex
    set_default_diff_ignore_regex
    get_default_tar_ignore_pattern
);

use Exporter qw(import);
use POSIX qw(:errno_h :sys_wait_h);
use Carp;
use File::Temp;
use File::Copy qw(cp);
use File::Basename;

use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Control;
use Dpkg::Checksums;
use Dpkg::Version;
use Dpkg::Compression;
use Dpkg::Path qw(check_files_are_the_same check_directory_traversal);
use Dpkg::Vendor qw(run_vendor_hook);
use Dpkg::Source::Format;
use Dpkg::OpenPGP;

my $diff_ignore_default_regex = '
# Ignore general backup files
(?:^|/).*~$|
# Ignore emacs recovery files
(?:^|/)\.#.*$|
# Ignore vi swap files
(?:^|/)\..*\.sw.$|
# Ignore baz-style junk files or directories
(?:^|/),,.*(?:$|/.*$)|
# File-names that should be ignored (never directories)
(?:^|/)(?:DEADJOE|\.arch-inventory|\.(?:bzr|cvs|hg|git|mtn-)ignore)$|
# File or directory names that should be ignored
(?:^|/)(?:CVS|RCS|\.deps|\{arch\}|\.arch-ids|\.svn|
\.hg(?:tags|sigs)?|_darcs|\.git(?:attributes|modules|review)?|
\.mailmap|\.shelf|_MTN|\.be|\.bzr(?:\.backup|tags)?)(?:$|/.*$)
';
# Take out comments and newlines
$diff_ignore_default_regex =~ s/^#.*$//mg;
$diff_ignore_default_regex =~ s/\n//sg;

no warnings 'qw'; ## no critic (TestingAndDebugging::ProhibitNoWarnings)
my @tar_ignore_default_pattern = qw(
*.a
*.la
*.o
*.so
.*.sw?
*/*~
,,*
.[#~]*
.arch-ids
.arch-inventory
.be
.bzr
.bzr.backup
.bzr.tags
.bzrignore
.cvsignore
.deps
.git
.gitattributes
.gitignore
.gitmodules
.gitreview
.hg
.hgignore
.hgsigs
.hgtags
.mailmap
.mtn-ignore
.shelf
.svn
CVS
DEADJOE
RCS
_MTN
_darcs
{arch}
);
## use critic

=head1 FUNCTIONS

=over 4

=item $string = get_default_diff_ignore_regex()

Returns the default diff ignore regex.

=cut

sub get_default_diff_ignore_regex {
    return $diff_ignore_default_regex;
}

=item set_default_diff_ignore_regex($string)

Set a regex as the new default diff ignore regex.

=cut

sub set_default_diff_ignore_regex {
    my $regex = shift;

    $diff_ignore_default_regex = $regex;
}

=item @array = get_default_tar_ignore_pattern()

Returns the default tar ignore pattern, as an array.

=cut

sub get_default_tar_ignore_pattern {
    return @tar_ignore_default_pattern;
}

=back

=head1 METHODS

=over 4

=item $p = Dpkg::Source::Package->new(%opts, options => {})

Creates a new object corresponding to a source package. When the key
B<filename> is set to a F<.dsc> file, it will be used to initialize the
source package with its description. Otherwise if the B<format> key is
set to a valid value, the object will be initialized for that format
(since dpkg 1.19.3).

The B<options> key is a hash ref which supports the following options:

=over 8

=item skip_debianization

If set to 1, do not apply Debian changes on the extracted source package.

=item skip_patches

If set to 1, do not apply Debian-specific patches. This options is
specific for source packages using format "2.0" and "3.0 (quilt)".

=item require_valid_signature

If set to 1, the check_signature() method will be stricter and will error
out if the signature can't be verified.

=item require_strong_checksums

If set to 1, the check_checksums() method will be stricter and will error
out if there is no strong checksum.

=item copy_orig_tarballs

If set to 1, the extraction will copy the upstream tarballs next the
target directory. This is useful if you want to be able to rebuild the
source package after its extraction.

=back

=cut

# Class methods
sub new {
    my ($this, %args) = @_;
    my $class = ref($this) || $this;
    my $self = {
        fields => Dpkg::Control->new(type => CTRL_PKG_SRC),
        format => Dpkg::Source::Format->new(),
        options => {},
        checksums => Dpkg::Checksums->new(),
    };
    bless $self, $class;
    if (exists $args{options}) {
        $self->{options} = $args{options};
    }
    if (exists $args{filename}) {
        $self->initialize($args{filename});
        $self->init_options();
    } elsif ($args{format}) {
        $self->{fields}{Format} = $args{format};
        $self->upgrade_object_type(0);
        $self->init_options();
    }
    return $self;
}

sub init_options {
    my $self = shift;
    # Use full ignore list by default
    # note: this function is not called by V1 packages
    $self->{options}{diff_ignore_regex} ||= $diff_ignore_default_regex;
    $self->{options}{diff_ignore_regex} .= '|(?:^|/)debian/source/local-.*$';
    $self->{options}{diff_ignore_regex} .= '|(?:^|/)debian/files(?:\.new)?$';
    if (defined $self->{options}{tar_ignore}) {
        $self->{options}{tar_ignore} = [ @tar_ignore_default_pattern ]
            unless @{$self->{options}{tar_ignore}};
    } else {
        $self->{options}{tar_ignore} = [ @tar_ignore_default_pattern ];
    }
    push @{$self->{options}{tar_ignore}},
         'debian/source/local-options',
         'debian/source/local-patch-header',
         'debian/files',
         'debian/files.new';
    $self->{options}{copy_orig_tarballs} //= 0;

    # Skip debianization while specific to some formats has an impact
    # on code common to all formats
    $self->{options}{skip_debianization} //= 0;
    $self->{options}{skip_patches} //= 0;

    # Set default validation checks.
    $self->{options}{require_valid_signature} //= 0;
    $self->{options}{require_strong_checksums} //= 0;

    # Set default compressor for new formats.
    $self->{options}{compression} //= 'xz';
    $self->{options}{comp_level} //= compression_get_property($self->{options}{compression},
                                                              'default_level');
    $self->{options}{comp_ext} //= compression_get_property($self->{options}{compression},
                                                            'file_ext');
}

sub initialize {
    my ($self, $filename) = @_;
    my ($fn, $dir) = fileparse($filename);
    error(g_('%s is not the name of a file'), $filename) unless $fn;
    $self->{basedir} = $dir || './';
    $self->{filename} = $fn;

    # Read the fields
    my $fields = $self->{fields};
    $fields->load($filename);
    $self->{is_signed} = $fields->get_option('is_pgp_signed');

    foreach my $f (qw(Source Version Files)) {
        unless (defined($fields->{$f})) {
            error(g_('missing critical source control field %s'), $f);
        }
    }

    $self->{checksums}->add_from_control($fields, use_files_for_md5 => 1);

    $self->upgrade_object_type(0);
}

sub upgrade_object_type {
    my ($self, $update_format) = @_;
    $update_format //= 1;

    my $format = $self->{fields}{'Format'} // '1.0';
    my ($major, $minor, $variant) = $self->{format}->set($format);

    my $module = "Dpkg::Source::Package::V$major";
    $module .= '::' . ucfirst $variant if defined $variant;
    eval qq{
        pop \@INC if \$INC[-1] eq '.';
        require $module;
        \$minor = \$${module}::CURRENT_MINOR_VERSION;
    };
    if ($@) {
        error(g_("source package format '%s' is not supported: %s"),
              $format, $@);
    }
    if ($update_format) {
        $self->{format}->set_from_parts($major, $minor, $variant);
        $self->{fields}{'Format'} = $self->{format}->get();
    }

    $module->prerequisites() if $module->can('prerequisites');
    bless $self, $module;
}

=item $p->get_filename()

Returns the filename of the DSC file.

=cut

sub get_filename {
    my $self = shift;
    return $self->{basedir} . $self->{filename};
}

=item $p->get_files()

Returns the list of files referenced by the source package. The filenames
usually do not have any path information.

=cut

sub get_files {
    my $self = shift;
    return $self->{checksums}->get_files();
}

=item $p->check_checksums()

Verify the checksums embedded in the DSC file. It requires the presence of
the other files constituting the source package. If any inconsistency is
discovered, it immediately errors out. It will make sure at least one strong
checksum is present.

If the object has been created with the "require_strong_checksums" option,
then any problem will result in a fatal error.

=cut

sub check_checksums {
    my $self = shift;
    my $checksums = $self->{checksums};
    my $warn_on_weak = 0;

    # add_from_file verify the checksums if they are already existing
    foreach my $file ($checksums->get_files()) {
        if (not $checksums->has_strong_checksums($file)) {
            if ($self->{options}{require_strong_checksums}) {
                error(g_('source package uses only weak checksums'));
            } else {
                $warn_on_weak = 1;
            }
        }
	$checksums->add_from_file($self->{basedir} . $file, key => $file);
    }

    warning(g_('source package uses only weak checksums')) if $warn_on_weak;
}

sub get_basename {
    my ($self, $with_revision) = @_;
    my $f = $self->{fields};
    unless (exists $f->{'Source'} and exists $f->{'Version'}) {
        error(g_('%s and %s fields are required to compute the source basename'),
              'Source', 'Version');
    }
    my $v = Dpkg::Version->new($f->{'Version'});
    my $vs = $v->as_string(omit_epoch => 1, omit_revision => !$with_revision);
    return $f->{'Source'} . '_' . $vs;
}

sub find_original_tarballs {
    my ($self, %opts) = @_;
    $opts{extension} //= compression_get_file_extension_regex();
    $opts{include_main} //= 1;
    $opts{include_supplementary} //= 1;
    my $basename = $self->get_basename();
    my @tar;
    foreach my $dir ('.', $self->{basedir}, $self->{options}{origtardir}) {
        next unless defined($dir) and -d $dir;
        opendir(my $dir_dh, $dir) or syserr(g_('cannot opendir %s'), $dir);
        push @tar, map { "$dir/$_" } grep {
		($opts{include_main} and
		 /^\Q$basename\E\.orig\.tar\.$opts{extension}$/) or
		($opts{include_supplementary} and
		 /^\Q$basename\E\.orig-[[:alnum:]-]+\.tar\.$opts{extension}$/)
	    } readdir($dir_dh);
        closedir($dir_dh);
    }
    return @tar;
}

=item $p->get_upstream_signing_key($dir)

Get the filename for the upstream key.

=cut

sub get_upstream_signing_key {
    my ($self, $dir) = @_;

    return "$dir/debian/upstream/signing-key.asc";
}

=item $p->check_original_tarball_signature($dir, @asc)

Verify the original upstream tarball signatures @asc using the upstream
public keys. It requires the origin upstream tarballs, their signatures
and the upstream signing key, as found in an unpacked source tree $dir.
If any inconsistency is discovered, it immediately errors out.

=cut

sub check_original_tarball_signature {
    my ($self, $dir, @asc) = @_;

    my $upstream_key = $self->get_upstream_signing_key($dir);
    if (not -e $upstream_key) {
        warning(g_('upstream tarball signatures but no upstream signing key'));
        return;
    }

    my $keyring = File::Temp->new(UNLINK => 1, SUFFIX => '.gpg');
    my %opts = (
        require_valid_signature => $self->{options}{require_valid_signature},
    );
    Dpkg::OpenPGP::import_key($upstream_key,
        %opts,
        keyring => $keyring,
    );

    foreach my $asc (@asc) {
        Dpkg::OpenPGP::verify_signature($asc,
            %opts,
            keyrings => [ $keyring ],
            datafile => $asc =~ s/\.asc$//r,
        );
    }
}

=item $bool = $p->is_signed()

Returns 1 if the DSC files contains an embedded OpenPGP signature.
Otherwise returns 0.

=cut

sub is_signed {
    my $self = shift;
    return $self->{is_signed};
}

=item $p->check_signature()

Implement the same OpenPGP signature check that dpkg-source does.
In case of problems, it prints a warning or errors out.

If the object has been created with the "require_valid_signature" option,
then any problem will result in a fatal error.

=cut

sub check_signature {
    my $self = shift;
    my $dsc = $self->get_filename();
    my @keyrings;

    if (length $ENV{HOME} and -r "$ENV{HOME}/.gnupg/trustedkeys.gpg") {
        push @keyrings, "$ENV{HOME}/.gnupg/trustedkeys.gpg";
    }
    foreach my $vendor_keyring (run_vendor_hook('package-keyrings')) {
        if (-r $vendor_keyring) {
            push @keyrings, $vendor_keyring;
        }
    }

    my %opts = (
        keyrings => \@keyrings,
        require_valid_signature => $self->{options}{require_valid_signature},
    );
    Dpkg::OpenPGP::verify_signature($dsc, %opts);
}

sub describe_cmdline_options {
    return;
}

sub parse_cmdline_options {
    my ($self, @opts) = @_;
    foreach my $option (@opts) {
        if (not $self->parse_cmdline_option($option)) {
            warning(g_('%s is not a valid option for %s'), $option, ref $self);
        }
    }
}

sub parse_cmdline_option {
    return 0;
}

=item $p->extract($targetdir)

Extracts the source package in the target directory $targetdir. Beware
that if $targetdir already exists, it will be erased (as long as the
no_overwrite_dir option is set).

=cut

sub extract {
    my ($self, $newdirectory) = @_;

    my ($ok, $error) = version_check($self->{fields}{'Version'});
    if (not $ok) {
        if ($self->{options}{ignore_bad_version}) {
            warning($error);
        } else {
            error($error);
        }
    }

    # Copy orig tarballs
    if ($self->{options}{copy_orig_tarballs}) {
        my $basename = $self->get_basename();
        my ($dirname, $destdir) = fileparse($newdirectory);
        $destdir ||= './';
	my $ext = compression_get_file_extension_regex();
        foreach my $orig (grep { /^\Q$basename\E\.orig(-[[:alnum:]-]+)?\.tar\.$ext$/ }
                          $self->get_files())
        {
            my $src = File::Spec->catfile($self->{basedir}, $orig);
            my $dst = File::Spec->catfile($destdir, $orig);
            if (not check_files_are_the_same($src, $dst, 1)) {
                cp($src, $dst)
                    or syserr(g_('cannot copy %s to %s'), $src, $dst);
            }
        }
    }

    # Try extract
    $self->do_extract($newdirectory);

    # Check for directory traversals.
    if (not $self->{options}{skip_debianization} and not $self->{no_check}) {
        # We need to add a trailing slash to handle the debian directory
        # possibly being a symlink.
        check_directory_traversal($newdirectory, "$newdirectory/debian/");
    }

    # Store format if non-standard so that next build keeps the same format
    if ($self->{fields}{'Format'} and
        $self->{fields}{'Format'} ne '1.0' and
        not $self->{options}{skip_debianization})
    {
        my $srcdir = File::Spec->catdir($newdirectory, 'debian', 'source');
        my $format_file = File::Spec->catfile($srcdir, 'format');
	unless (-e $format_file) {
	    mkdir($srcdir) unless -e $srcdir;
            $self->{format}->save($format_file);
	}
    }

    # Make sure debian/rules is executable
    my $rules = File::Spec->catfile($newdirectory, 'debian', 'rules');
    my @s = lstat($rules);
    if (not scalar(@s)) {
        unless ($! == ENOENT) {
            syserr(g_('cannot stat %s'), $rules);
        }
        warning(g_('%s does not exist'), $rules)
            unless $self->{options}{skip_debianization};
    } elsif (-f _) {
        chmod($s[2] | 0111, $rules)
            or syserr(g_('cannot make %s executable'), $rules);
    } else {
        warning(g_('%s is not a plain file'), $rules);
    }
}

sub do_extract {
    croak 'Dpkg::Source::Package does not know how to unpack a ' .
          'source package; use one of the subclasses';
}

# Function used specifically during creation of a source package

sub before_build {
    my ($self, $dir) = @_;
}

sub build {
    my $self = shift;

    $self->do_build(@_);
}

sub after_build {
    my ($self, $dir) = @_;
}

sub do_build {
    croak 'Dpkg::Source::Package does not know how to build a ' .
          'source package; use one of the subclasses';
}

sub can_build {
    my ($self, $dir) = @_;
    return (0, 'can_build() has not been overridden');
}

sub add_file {
    my ($self, $filename) = @_;
    my ($fn, $dir) = fileparse($filename);
    if ($self->{checksums}->has_file($fn)) {
        croak "tried to add file '$fn' twice";
    }
    $self->{checksums}->add_from_file($filename, key => $fn);
    $self->{checksums}->export_to_control($self->{fields},
					    use_files_for_md5 => 1);
}

sub commit {
    my $self = shift;

    $self->do_commit(@_);
}

sub do_commit {
    my ($self, $dir) = @_;
    info(g_("'%s' is not supported by the source format '%s'"),
         'dpkg-source --commit', $self->{fields}{'Format'});
}

sub write_dsc {
    my ($self, %opts) = @_;
    my $fields = $self->{fields};

    foreach my $f (keys %{$opts{override}}) {
	$fields->{$f} = $opts{override}{$f};
    }

    unless ($opts{nocheck}) {
        foreach my $f (qw(Source Version Architecture)) {
            unless (defined($fields->{$f})) {
                error(g_('missing information for critical output field %s'), $f);
            }
        }
        foreach my $f (qw(Maintainer Standards-Version)) {
            unless (defined($fields->{$f})) {
                warning(g_('missing information for output field %s'), $f);
            }
        }
    }

    foreach my $f (keys %{$opts{remove}}) {
	delete $fields->{$f};
    }

    my $filename = $opts{filename};
    $filename //= $self->get_basename(1) . '.dsc';
    open(my $dsc_fh, '>', $filename)
        or syserr(g_('cannot write %s'), $filename);
    $fields->apply_substvars($opts{substvars});
    $fields->output($dsc_fh);
    close($dsc_fh);
}

=back

=head1 CHANGES

=head2 Version 2.01 (dpkg 1.20.1)

New method: get_upstream_signing_key().

=head2 Version 2.00 (dpkg 1.20.0)

New method: check_original_tarball_signature().

Remove variable: $diff_ignore_default_regexp.

Hide variable: @tar_ignore_default_pattern.

=head2 Version 1.03 (dpkg 1.19.3)

New option: format in new().

=head2 Version 1.02 (dpkg 1.18.7)

New option: require_strong_checksums in check_checksums().

=head2 Version 1.01 (dpkg 1.17.2)

New functions: get_default_diff_ignore_regex(), set_default_diff_ignore_regex(),
get_default_tar_ignore_pattern()

Deprecated variables: $diff_ignore_default_regexp, @tar_ignore_default_pattern

=head2 Version 1.00 (dpkg 1.16.1)

Mark the module as public.

=cut

1;
