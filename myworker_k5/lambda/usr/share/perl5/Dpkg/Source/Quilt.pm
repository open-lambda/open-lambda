# Copyright © 2008-2012 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::Source::Quilt;

use strict;
use warnings;

our $VERSION = '0.02';

use List::Util qw(any none);
use File::Spec;
use File::Copy;
use File::Find;
use File::Path qw(make_path);
use File::Basename;

use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Source::Patch;
use Dpkg::Source::Functions qw(erasedir chmod_if_needed fs_time);
use Dpkg::Vendor qw(get_current_vendor);

sub new {
    my ($this, $dir, %opts) = @_;
    my $class = ref($this) || $this;

    my $self = {
        dir => $dir,
    };
    bless $self, $class;

    $self->load_series();
    $self->load_db();

    return $self;
}

sub setup_db {
    my $self = shift;
    my $db_dir = $self->get_db_file();
    if (not -d $db_dir) {
        mkdir $db_dir or syserr(g_('cannot mkdir %s'), $db_dir);
    }
    my $file = $self->get_db_file('.version');
    if (not -e $file) {
        open(my $version_fh, '>', $file) or syserr(g_('cannot write %s'), $file);
        print { $version_fh } "2\n";
        close($version_fh);
    }
    # The files below are used by quilt to know where patches are stored
    # and what file contains the patch list (supported by quilt >= 0.48-5
    # in Debian).
    $file = $self->get_db_file('.quilt_patches');
    if (not -e $file) {
        open(my $qpatch_fh, '>', $file) or syserr(g_('cannot write %s'), $file);
        print { $qpatch_fh } "debian/patches\n";
        close($qpatch_fh);
    }
    $file = $self->get_db_file('.quilt_series');
    if (not -e $file) {
        open(my $qseries_fh, '>', $file) or syserr(g_('cannot write %s'), $file);
        my $series = $self->get_series_file();
        $series = (File::Spec->splitpath($series))[2];
        print { $qseries_fh } "$series\n";
        close($qseries_fh);
    }
}

sub load_db {
    my $self = shift;

    my $pc_applied = $self->get_db_file('applied-patches');
    $self->{applied_patches} = [ $self->read_patch_list($pc_applied) ];
}

sub save_db {
    my $self = shift;

    $self->setup_db();
    my $pc_applied = $self->get_db_file('applied-patches');
    $self->write_patch_list($pc_applied, $self->{applied_patches});
}

sub load_series {
    my ($self, %opts) = @_;

    my $series = $self->get_series_file();
    $self->{series} = [ $self->read_patch_list($series, %opts) ];
}

sub series {
    my $self = shift;
    return @{$self->{series}};
}

sub applied {
    my $self = shift;
    return @{$self->{applied_patches}};
}

sub top {
    my $self = shift;
    my $count = scalar @{$self->{applied_patches}};
    return $self->{applied_patches}[$count - 1] if $count;
    return;
}

sub register {
    my ($self, $patch_name) = @_;

    return if any { $_ eq $patch_name } @{$self->{series}};

    # Add patch to series files.
    $self->setup_db();
    $self->_file_add_line($self->get_series_file(), $patch_name);
    $self->_file_add_line($self->get_db_file('applied-patches'), $patch_name);
    $self->load_db();
    $self->load_series();

    # Ensure quilt meta-data is created and in sync with some trickery:
    # Reverse-apply the patch, drop .pc/$patch, and re-apply it with the
    # correct options to recreate the backup files.
    $self->pop(reverse_apply => 1);
    $self->push();
}

sub unregister {
    my ($self, $patch_name) = @_;

    return if none { $_ eq $patch_name } @{$self->{series}};

    my $series = $self->get_series_file();

    $self->_file_drop_line($series, $patch_name);
    $self->_file_drop_line($self->get_db_file('applied-patches'), $patch_name);
    erasedir($self->get_db_file($patch_name));
    $self->load_db();
    $self->load_series();

    # Clean up empty series.
    unlink $series if -z $series;
}

sub next {
    my $self = shift;
    my $count_applied = scalar @{$self->{applied_patches}};
    my $count_series = scalar @{$self->{series}};
    return $self->{series}[$count_applied] if ($count_series > $count_applied);
    return;
}

sub push {
    my ($self, %opts) = @_;
    $opts{verbose} //= 0;
    $opts{timestamp} //= fs_time($self->{dir});

    my $patch = $self->next();
    return unless defined $patch;

    my $path = $self->get_patch_file($patch);
    my $obj = Dpkg::Source::Patch->new(filename => $path);

    info(g_('applying %s'), $patch) if $opts{verbose};
    eval {
        $obj->apply($self->{dir}, timestamp => $opts{timestamp},
                    verbose => $opts{verbose},
                    force_timestamp => 1, create_dirs => 1, remove_backup => 0,
                    options => [ '-t', '-F', '0', '-N', '-p1', '-u',
                                 '-V', 'never', '-E', '-b',
                                 '-B', ".pc/$patch/", '--reject-file=-' ]);
    };
    if ($@) {
        info(g_('the patch has fuzz which is not allowed, or is malformed'));
        info(g_("if patch '%s' is correctly applied by quilt, use '%s' to update it"),
             $patch, 'quilt refresh');
        info(g_('if the file is present in the unpacked source, make sure it ' .
                'is also present in the orig tarball'));
        $self->restore_quilt_backup_files($patch, %opts);
        erasedir($self->get_db_file($patch));
        die $@;
    }
    CORE::push @{$self->{applied_patches}}, $patch;
    $self->save_db();
}

sub pop {
    my ($self, %opts) = @_;
    $opts{verbose} //= 0;
    $opts{timestamp} //= fs_time($self->{dir});
    $opts{reverse_apply} //= 0;

    my $patch = $self->top();
    return unless defined $patch;

    info(g_('unapplying %s'), $patch) if $opts{verbose};
    my $backup_dir = $self->get_db_file($patch);
    if (-d $backup_dir and not $opts{reverse_apply}) {
        # Use the backup copies to restore
        $self->restore_quilt_backup_files($patch);
    } else {
        # Otherwise reverse-apply the patch
        my $path = $self->get_patch_file($patch);
        my $obj = Dpkg::Source::Patch->new(filename => $path);

        $obj->apply($self->{dir}, timestamp => $opts{timestamp},
                    verbose => 0, force_timestamp => 1, remove_backup => 0,
                    options => [ '-R', '-t', '-N', '-p1',
                                 '-u', '-V', 'never', '-E',
                                 '--no-backup-if-mismatch' ]);
    }

    erasedir($backup_dir);
    pop @{$self->{applied_patches}};
    $self->save_db();
}

sub get_db_version {
    my $self = shift;
    my $pc_ver = $self->get_db_file('.version');
    if (-f $pc_ver) {
        open(my $ver_fh, '<', $pc_ver) or syserr(g_('cannot read %s'), $pc_ver);
        my $version = <$ver_fh>;
        chomp $version;
        close($ver_fh);
        return $version;
    }
    return;
}

sub find_problems {
    my $self = shift;
    my $patch_dir = $self->get_patch_file();
    if (-e $patch_dir and not -d _) {
        return sprintf(g_('%s should be a directory or non-existing'), $patch_dir);
    }
    my $series = $self->get_series_file();
    if (-e $series and not -f _) {
        return sprintf(g_('%s should be a file or non-existing'), $series);
    }
    return;
}

sub get_series_file {
    my $self = shift;
    my $vendor = lc(get_current_vendor() || 'debian');
    # Series files are stored alongside patches
    my $default_series = $self->get_patch_file('series');
    my $vendor_series = $self->get_patch_file("$vendor.series");
    return $vendor_series if -e $vendor_series;
    return $default_series;
}

sub get_db_file {
    my $self = shift;
    return File::Spec->catfile($self->{dir}, '.pc', @_);
}

sub get_db_dir {
    my $self = shift;
    return $self->get_db_file();
}

sub get_patch_file {
    my $self = shift;
    return File::Spec->catfile($self->{dir}, 'debian', 'patches', @_);
}

sub get_patch_dir {
    my $self = shift;
    return $self->get_patch_file();
}

## METHODS BELOW ARE INTERNAL ##

sub _file_load {
    my ($self, $file) = @_;

    open my $file_fh, '<', $file or syserr(g_('cannot read %s'), $file);
    my @lines = <$file_fh>;
    close $file_fh;

    return @lines;
}

sub _file_add_line {
    my ($self, $file, $line) = @_;

    my @lines;
    @lines = $self->_file_load($file) if -f $file;
    CORE::push @lines, $line;
    chomp @lines;

    open my $file_fh, '>', $file or syserr(g_('cannot write %s'), $file);
    print { $file_fh } "$_\n" foreach @lines;
    close $file_fh;
}

sub _file_drop_line {
    my ($self, $file, $re) = @_;

    my @lines = $self->_file_load($file);
    open my $file_fh, '>', $file or syserr(g_('cannot write %s'), $file);
    print { $file_fh } $_ foreach grep { not /^\Q$re\E\s*$/ } @lines;
    close $file_fh;
}

sub read_patch_list {
    my ($self, $file, %opts) = @_;
    return () if not defined $file or not -f $file;
    $opts{warn_options} //= 0;
    my @patches;
    open(my $series_fh, '<' , $file) or syserr(g_('cannot read %s'), $file);
    while (defined(my $line = <$series_fh>)) {
        chomp $line;
        # Strip leading/trailing spaces
        $line =~ s/^\s+//;
        $line =~ s/\s+$//;
        # Strip comment
        $line =~ s/(?:^|\s+)#.*$//;
        next unless $line;
        if ($line =~ /^(\S+)\s+(.*)$/) {
            $line = $1;
            if ($2 ne '-p1') {
                warning(g_('the series file (%s) contains unsupported ' .
                           "options ('%s', line %s); dpkg-source might " .
                           'fail when applying patches'),
                        $file, $2, $.) if $opts{warn_options};
            }
        }
        if ($line =~ m{(^|/)\.\./}) {
            error(g_('%s contains an insecure path: %s'), $file, $line);
        }
        CORE::push @patches, $line;
    }
    close($series_fh);
    return @patches;
}

sub write_patch_list {
    my ($self, $series, $patches) = @_;

    open my $series_fh, '>', $series or syserr(g_('cannot write %s'), $series);
    foreach my $patch (@{$patches}) {
        print { $series_fh } "$patch\n";
    }
    close $series_fh;
}

sub restore_quilt_backup_files {
    my ($self, $patch, %opts) = @_;
    my $patch_dir = $self->get_db_file($patch);
    return unless -d $patch_dir;
    info(g_('restoring quilt backup files for %s'), $patch) if $opts{verbose};
    find({
        no_chdir => 1,
        wanted => sub {
            return if -d;
            my $relpath_in_srcpkg = File::Spec->abs2rel($_, $patch_dir);
            my $target = File::Spec->catfile($self->{dir}, $relpath_in_srcpkg);
            if (-s) {
                unlink($target);
                make_path(dirname($target));
                unless (link($_, $target)) {
                    copy($_, $target)
                        or syserr(g_('failed to copy %s to %s'), $_, $target);
                    chmod_if_needed((stat _)[2], $target)
                        or syserr(g_("unable to change permission of '%s'"), $target);
                }
            } else {
                # empty files are "backups" for new files that patch created
                unlink($target);
            }
        }
    }, $patch_dir);
}

1;
