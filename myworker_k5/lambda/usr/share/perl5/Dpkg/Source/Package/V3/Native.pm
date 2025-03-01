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

package Dpkg::Source::Package::V3::Native;

use strict;
use warnings;

our $VERSION = '0.01';

use Cwd;
use File::Basename;
use File::Temp qw(tempfile);

use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Compression;
use Dpkg::Exit qw(push_exit_handler pop_exit_handler);
use Dpkg::Version;
use Dpkg::Source::Archive;
use Dpkg::Source::Functions qw(erasedir);

use parent qw(Dpkg::Source::Package);

our $CURRENT_MINOR_VERSION = '0';

sub do_extract {
    my ($self, $newdirectory) = @_;
    my $sourcestyle = $self->{options}{sourcestyle};
    my $fields = $self->{fields};

    my $dscdir = $self->{basedir};
    my $basename = $self->get_basename();
    my $basenamerev = $self->get_basename(1);

    my $tarfile;
    my $comp_ext_regex = compression_get_file_extension_regex();
    foreach my $file ($self->get_files()) {
	if ($file =~ /^\Q$basenamerev\E\.tar\.$comp_ext_regex$/) {
            error(g_('multiple tarfiles in native source package')) if $tarfile;
            $tarfile = $file;
	} else {
	    error(g_('unrecognized file for a native source package: %s'), $file);
	}
    }

    error(g_('no tarfile in Files field')) unless $tarfile;

    if ($self->{options}{no_overwrite_dir} and -e $newdirectory) {
        error(g_('unpack target exists: %s'), $newdirectory);
    } else {
        erasedir($newdirectory);
    }

    info(g_('unpacking %s'), $tarfile);
    my $tar = Dpkg::Source::Archive->new(filename => "$dscdir$tarfile");
    $tar->extract($newdirectory);
}

sub can_build {
    my ($self, $dir) = @_;

    my $v = Dpkg::Version->new($self->{fields}->{'Version'});
    warning (g_('native package version may not have a revision'))
        unless $v->is_native();

    return 1;
}

sub do_build {
    my ($self, $dir) = @_;
    my @tar_ignore = map { "--exclude=$_" } @{$self->{options}{tar_ignore}};
    my @argv = @{$self->{options}{ARGV}};

    if (scalar(@argv)) {
        usageerr(g_("-b takes only one parameter with format '%s'"),
                 $self->{fields}{'Format'});
    }

    my $sourcepackage = $self->{fields}{'Source'};
    my $basenamerev = $self->get_basename(1);
    my $tarname = "$basenamerev.tar." . $self->{options}{comp_ext};

    info(g_('building %s in %s'), $sourcepackage, $tarname);

    my ($ntfh, $newtar) = tempfile("$tarname.new.XXXXXX",
                                   DIR => getcwd(), UNLINK => 0);
    push_exit_handler(sub { unlink($newtar) });

    my ($dirname, $dirbase) = fileparse($dir);
    my $tar = Dpkg::Source::Archive->new(filename => $newtar,
                compression => compression_guess_from_filename($tarname),
                compression_level => $self->{options}{comp_level});
    $tar->create(options => \@tar_ignore, chdir => $dirbase);
    $tar->add_directory($dirname);
    $tar->finish();
    rename($newtar, $tarname)
        or syserr(g_("unable to rename '%s' (newly created) to '%s'"),
                  $newtar, $tarname);
    pop_exit_handler();
    chmod(0666 &~ umask(), $tarname)
        or syserr(g_("unable to change permission of '%s'"), $tarname);

    $self->add_file($tarname);
}

1;
