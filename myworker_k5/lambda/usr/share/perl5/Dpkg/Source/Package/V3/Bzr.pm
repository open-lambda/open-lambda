#
# bzr support for dpkg-source
#
# Copyright © 2007 Colin Watson <cjwatson@debian.org>.
# Based on Dpkg::Source::Package::V3_0::git, which is:
# Copyright © 2007 Joey Hess <joeyh@debian.org>.
# Copyright © 2008 Frank Lichtenheld <djpig@debian.org>
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

package Dpkg::Source::Package::V3::Bzr;

use strict;
use warnings;

our $VERSION = '0.01';

use Cwd;
use File::Basename;
use File::Find;
use File::Temp qw(tempdir);

use Dpkg::Gettext;
use Dpkg::Compression;
use Dpkg::ErrorHandling;
use Dpkg::Source::Archive;
use Dpkg::Exit qw(push_exit_handler pop_exit_handler);
use Dpkg::Path qw(find_command);
use Dpkg::Source::Functions qw(erasedir);

use parent qw(Dpkg::Source::Package);

our $CURRENT_MINOR_VERSION = '0';

sub prerequisites {
    return 1 if find_command('bzr');
    error(g_('cannot unpack bzr-format source package because ' .
             'bzr is not in the PATH'));
}

sub _sanity_check {
    my $srcdir = shift;

    if (! -d "$srcdir/.bzr") {
        error(g_('source directory is not the top directory of a bzr repository (%s/.bzr not present), but Format bzr was specified'),
              $srcdir);
    }

    # Symlinks from .bzr to outside could cause unpack failures, or
    # point to files they shouldn't, so check for and don't allow.
    if (-l "$srcdir/.bzr") {
        error(g_('%s is a symlink'), "$srcdir/.bzr");
    }
    my $abs_srcdir = Cwd::abs_path($srcdir);
    find(sub {
        if (-l) {
            if (Cwd::abs_path(readlink) !~ /^\Q$abs_srcdir\E(?:\/|$)/) {
                error(g_('%s is a symlink to outside %s'),
                      $File::Find::name, $srcdir);
            }
        }
    }, "$srcdir/.bzr");

    return 1;
}

sub can_build {
    my ($self, $dir) = @_;

    return (0, g_("doesn't contain a bzr repository")) unless -d "$dir/.bzr";
    return 1;
}

sub do_build {
    my ($self, $dir) = @_;
    my @argv = @{$self->{options}{ARGV}};
    # TODO: warn here?
    #my @tar_ignore = map { "--exclude=$_" } @{$self->{options}{tar_ignore}};
    my $diff_ignore_regex = $self->{options}{diff_ignore_regex};

    $dir =~ s{/+$}{}; # Strip trailing /
    my ($dirname, $updir) = fileparse($dir);

    if (scalar(@argv)) {
        usageerr(g_("-b takes only one parameter with format '%s'"),
                 $self->{fields}{'Format'});
    }

    my $sourcepackage = $self->{fields}{'Source'};
    my $basenamerev = $self->get_basename(1);
    my $basename = $self->get_basename();
    my $basedirname = $basename;
    $basedirname =~ s/_/-/;

    _sanity_check($dir);

    my $old_cwd = getcwd();
    chdir $dir or syserr(g_("unable to chdir to '%s'"), $dir);

    local $_;

    # Check for uncommitted files.
    # To support dpkg-source -i, remove any ignored files from the
    # output of bzr status.
    open(my $bzr_status_fh, '-|', 'bzr', 'status')
        or subprocerr('bzr status');
    my @files;
    while (<$bzr_status_fh>) {
        chomp;
        next unless s/^ +//;
        if (! length $diff_ignore_regex ||
            ! m/$diff_ignore_regex/o) {
            push @files, $_;
        }
    }
    close($bzr_status_fh) or syserr(g_('bzr status exited nonzero'));
    if (@files) {
        error(g_('uncommitted, not-ignored changes in working directory: %s'),
              join(' ', @files));
    }

    chdir $old_cwd or syserr(g_("unable to chdir to '%s'"), $old_cwd);

    my $tmp = tempdir("$dirname.bzr.XXXXXX", DIR => $updir);
    push_exit_handler(sub { erasedir($tmp) });
    my $tardir = "$tmp/$dirname";

    system('bzr', 'branch', $dir, $tardir);
    subprocerr("bzr branch $dir $tardir") if $?;

    # Remove the working tree.
    system('bzr', 'remove-tree', $tardir);
    subprocerr("bzr remove-tree $tardir") if $?;

    # Some branch metadata files are unhelpful.
    unlink("$tardir/.bzr/branch/branch-name",
           "$tardir/.bzr/branch/parent");

    # Create the tar file
    my $debianfile = "$basenamerev.bzr.tar." . $self->{options}{comp_ext};
    info(g_('building %s in %s'),
         $sourcepackage, $debianfile);
    my $tar = Dpkg::Source::Archive->new(filename => $debianfile,
                                         compression => $self->{options}{compression},
                                         compression_level => $self->{options}{comp_level});
    $tar->create(chdir => $tmp);
    $tar->add_directory($dirname);
    $tar->finish();

    erasedir($tmp);
    pop_exit_handler();

    $self->add_file($debianfile);
}

# Called after a tarball is unpacked, to check out the working copy.
sub do_extract {
    my ($self, $newdirectory) = @_;
    my $fields = $self->{fields};

    my $dscdir = $self->{basedir};

    my $basename = $self->get_basename();
    my $basenamerev = $self->get_basename(1);

    my @files = $self->get_files();
    if (@files > 1) {
        error(g_('format v3.0 (bzr) uses only one source file'));
    }
    my $tarfile = $files[0];
    my $comp_ext_regex = compression_get_file_extension_regex();
    if ($tarfile !~ /^\Q$basenamerev\E\.bzr\.tar\.$comp_ext_regex$/) {
        error(g_('expected %s, got %s'),
              "$basenamerev.bzr.tar.$comp_ext_regex", $tarfile);
    }

    if ($self->{options}{no_overwrite_dir} and -e $newdirectory) {
        error(g_('unpack target exists: %s'), $newdirectory);
    } else {
        erasedir($newdirectory);
    }

    # Extract main tarball
    info(g_('unpacking %s'), $tarfile);
    my $tar = Dpkg::Source::Archive->new(filename => "$dscdir$tarfile");
    $tar->extract($newdirectory);

    _sanity_check($newdirectory);

    my $old_cwd = getcwd();
    chdir($newdirectory)
        or syserr(g_("unable to chdir to '%s'"), $newdirectory);

    # Reconstitute the working tree.
    system('bzr', 'checkout');
    subprocerr('bzr checkout') if $?;

    chdir $old_cwd or syserr(g_("unable to chdir to '%s'"), $old_cwd);
}

1;
