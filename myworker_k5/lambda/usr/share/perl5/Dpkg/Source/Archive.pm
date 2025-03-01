# Copyright © 2008 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::Source::Archive;

use strict;
use warnings;

our $VERSION = '0.01';

use Carp;
use Errno qw(ENOENT);
use File::Temp qw(tempdir);
use File::Basename qw(basename);
use File::Spec;
use File::Find;
use Cwd;

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::IPC;
use Dpkg::Source::Functions qw(erasedir fixperms);

use parent qw(Dpkg::Compression::FileHandle);

sub create {
    my ($self, %opts) = @_;
    $opts{options} //= [];
    my %spawn_opts;
    # Possibly run tar from another directory
    if ($opts{chdir}) {
        $spawn_opts{chdir} = $opts{chdir};
        *$self->{chdir} = $opts{chdir};
    }
    # Redirect input/output appropriately
    $self->ensure_open('w');
    $spawn_opts{to_handle} = $self->get_filehandle();
    $spawn_opts{from_pipe} = \*$self->{tar_input};
    # Try to use a deterministic mtime.
    my $mtime = $opts{source_date} // $ENV{SOURCE_DATE_EPOCH} || time;
    # Call tar creation process
    $spawn_opts{delete_env} = [ 'TAR_OPTIONS' ];
    $spawn_opts{exec} = [ $Dpkg::PROGTAR, '-cf', '-', '--format=gnu', '--sort=name',
                          '--mtime', "\@$mtime", '--clamp-mtime', '--null',
                          '--numeric-owner', '--owner=0', '--group=0',
                          @{$opts{options}}, '-T', '-' ];
    *$self->{pid} = spawn(%spawn_opts);
    *$self->{cwd} = getcwd();
}

sub _add_entry {
    my ($self, $file) = @_;
    my $cwd = *$self->{cwd};
    croak 'call create() first' unless *$self->{tar_input};
    $file = $2 if ($file =~ /^\Q$cwd\E\/(.+)$/); # Relative names
    print({ *$self->{tar_input} } "$file\0")
        or syserr(g_('write on tar input'));
}

sub add_file {
    my ($self, $file) = @_;
    my $testfile = $file;
    if (*$self->{chdir}) {
        $testfile = File::Spec->catfile(*$self->{chdir}, $file);
    }
    croak 'add_file() does not handle directories'
        if not -l $testfile and -d _;
    $self->_add_entry($file);
}

sub add_directory {
    my ($self, $file) = @_;
    my $testfile = $file;
    if (*$self->{chdir}) {
        $testfile = File::Spec->catdir(*$self->{chdir}, $file);
    }
    croak 'add_directory() only handles directories'
        if -l $testfile or not -d _;
    $self->_add_entry($file);
}

sub finish {
    my $self = shift;

    close(*$self->{tar_input}) or syserr(g_('close on tar input'));
    wait_child(*$self->{pid}, cmdline => 'tar -cf -');
    delete *$self->{pid};
    delete *$self->{tar_input};
    delete *$self->{cwd};
    delete *$self->{chdir};
    $self->close();
}

sub extract {
    my ($self, $dest, %opts) = @_;
    $opts{options} //= [];
    $opts{in_place} //= 0;
    $opts{no_fixperms} //= 0;
    my %spawn_opts = (wait_child => 1);

    # Prepare destination
    my $template = basename($self->get_filename()) .  '.tmp-extract.XXXXX';
    unless (-e $dest) {
        # Kludge so that realpath works
        mkdir($dest) or syserr(g_('cannot create directory %s'), $dest);
    }
    my $tmp = tempdir($template, DIR => Cwd::realpath("$dest/.."), CLEANUP => 1);
    $spawn_opts{chdir} = $tmp;

    # Prepare stuff that handles the input of tar
    $self->ensure_open('r', delete_sig => [ 'PIPE' ]);
    $spawn_opts{from_handle} = $self->get_filehandle();

    # Call tar extraction process
    $spawn_opts{delete_env} = [ 'TAR_OPTIONS' ];
    $spawn_opts{exec} = [ $Dpkg::PROGTAR, '-xf', '-', '--no-same-permissions',
                          '--no-same-owner', @{$opts{options}} ];
    spawn(%spawn_opts);
    $self->close();

    # Fix permissions on extracted files because tar insists on applying
    # our umask _to the original permissions_ rather than mostly-ignoring
    # the original permissions.
    # We still need --no-same-permissions because otherwise tar might
    # extract directory setgid (which we want inherited, not
    # extracted); we need --no-same-owner because putting the owner
    # back is tedious - in particular, correct group ownership would
    # have to be calculated using mount options and other madness.
    fixperms($tmp) unless $opts{no_fixperms};

    # If we are extracting "in-place" do not remove the destination directory.
    if ($opts{in_place}) {
        my $canon_basedir = Cwd::realpath($dest);
        # On Solaris /dev/null points to /devices/pseudo/mm@0:null.
        my $canon_devnull = Cwd::realpath('/dev/null');
        my $check_symlink = sub {
            my $pathname = shift;
            my $canon_pathname = Cwd::realpath($pathname);
            if (not defined $canon_pathname) {
                return if $! == ENOENT;

                syserr(g_("pathname '%s' cannot be canonicalized"), $pathname);
            }
            return if $canon_pathname eq $canon_devnull;
            return if $canon_pathname eq $canon_basedir;
            return if $canon_pathname =~ m{^\Q$canon_basedir/\E};
            warning(g_("pathname '%s' points outside source root (to '%s')"),
                    $pathname, $canon_pathname);
        };

        my $move_in_place = sub {
            my $relpath = File::Spec->abs2rel($File::Find::name, $tmp);
            my $destpath = File::Spec->catfile($dest, $relpath);

            my ($mode, $atime, $mtime);
            lstat $File::Find::name
                or syserr(g_('cannot get source pathname %s metadata'), $File::Find::name);
            ((undef) x 2, $mode, (undef) x 5, $atime, $mtime) = lstat _;
            my $src_is_dir = -d _;

            my $dest_exists = 1;
            if (not lstat $destpath) {
                if ($! == ENOENT) {
                    $dest_exists = 0;
                } else {
                    syserr(g_('cannot get target pathname %s metadata'), $destpath);
                }
            }
            my $dest_is_dir = -d _;
            if ($dest_exists) {
                if ($dest_is_dir && $src_is_dir) {
                    # Refresh the destination directory attributes with the
                    # ones from the tarball.
                    chmod $mode, $destpath
                        or syserr(g_('cannot change directory %s mode'), $File::Find::name);
                    utime $atime, $mtime, $destpath
                        or syserr(g_('cannot change directory %s times'), $File::Find::name);

                    # We should do nothing, and just walk further tree.
                    return;
                } elsif ($dest_is_dir) {
                    rmdir $destpath
                        or syserr(g_('cannot remove destination directory %s'), $destpath);
                } else {
                    $check_symlink->($destpath);
                    unlink $destpath
                        or syserr(g_('cannot remove destination file %s'), $destpath);
                }
            }
            # If we are moving a directory, we do not need to walk it.
            if ($src_is_dir) {
                $File::Find::prune = 1;
            }
            rename $File::Find::name, $destpath
                or syserr(g_('cannot move %s to %s'), $File::Find::name, $destpath);
        };

        find({
            wanted => $move_in_place,
            no_chdir => 1,
            dangling_symlinks => 0,
        }, $tmp);
    } else {
        # Rename extracted directory
        opendir(my $dir_dh, $tmp) or syserr(g_('cannot opendir %s'), $tmp);
        my @entries = grep { $_ ne '.' && $_ ne '..' } readdir($dir_dh);
        closedir($dir_dh);

        erasedir($dest);

        if (scalar(@entries) == 1 && ! -l "$tmp/$entries[0]" && -d _) {
            rename("$tmp/$entries[0]", $dest)
                or syserr(g_('unable to rename %s to %s'),
                          "$tmp/$entries[0]", $dest);
        } else {
            rename($tmp, $dest)
                or syserr(g_('unable to rename %s to %s'), $tmp, $dest);
        }
    }
    erasedir($tmp);
}

1;
