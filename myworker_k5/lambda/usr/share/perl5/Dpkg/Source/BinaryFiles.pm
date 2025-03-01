# Copyright © 2008-2011 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2008-2015 Guillem Jover <guillem@debian.org>
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

package Dpkg::Source::BinaryFiles;

use strict;
use warnings;

our $VERSION = '0.01';

use Cwd;
use File::Path qw(make_path);
use File::Spec;
use File::Find;

use Dpkg::ErrorHandling;
use Dpkg::Gettext;
use Dpkg::Source::Functions qw(is_binary);

sub new {
    my ($this, $dir) = @_;
    my $class = ref($this) || $this;

    my $self = {
        dir => $dir,
        allowed_binaries => {},
        seen_binaries => {},
        include_binaries_path =>
            File::Spec->catfile($dir, 'debian', 'source', 'include-binaries'),
    };
    bless $self, $class;
    $self->load_allowed_binaries();
    return $self;
}

sub new_binary_found {
    my ($self, $path) = @_;

    $self->{seen_binaries}{$path} = 1;
}

sub load_allowed_binaries {
    my $self = shift;
    my $incbin_file = $self->{include_binaries_path};

    if (-f $incbin_file) {
        open my $incbin_fh, '<', $incbin_file
            or syserr(g_('cannot read %s'), $incbin_file);
        while (<$incbin_fh>) {
            chomp;
            s/^\s*//;
            s/\s*$//;
            next if /^#/ or length == 0;
            $self->{allowed_binaries}{$_} = 1;
        }
        close $incbin_fh;
    }
}

sub binary_is_allowed {
    my ($self, $path) = @_;

    return 1 if exists $self->{allowed_binaries}{$path};
    return 0;
}

sub update_debian_source_include_binaries {
    my $self = shift;

    my @unknown_binaries = $self->get_unknown_binaries();
    return unless scalar @unknown_binaries;

    my $incbin_file = $self->{include_binaries_path};
    make_path(File::Spec->catdir($self->{dir}, 'debian', 'source'));
    open my $incbin_fh, '>>', $incbin_file
        or syserr(g_('cannot write %s'), $incbin_file);
    foreach my $binary (@unknown_binaries) {
        print { $incbin_fh } "$binary\n";
        info(g_('adding %s to %s'), $binary, 'debian/source/include-binaries');
        $self->{allowed_binaries}{$binary} = 1;
    }
    close $incbin_fh;
}

sub get_unknown_binaries {
    my $self = shift;

    return grep { not $self->binary_is_allowed($_) } $self->get_seen_binaries();
}

sub get_seen_binaries {
    my $self = shift;
    my @seen = sort keys %{$self->{seen_binaries}};

    return @seen;
}

sub detect_binary_files {
    my ($self, %opts) = @_;

    my $unwanted_binaries = 0;
    my $check_binary = sub {
        if (-f and is_binary($_)) {
            my $fn = File::Spec->abs2rel($_, $self->{dir});
            $self->new_binary_found($fn);
            unless ($opts{include_binaries} or $self->binary_is_allowed($fn)) {
                errormsg(g_('unwanted binary file: %s'), $fn);
                $unwanted_binaries++;
            }
        }
    };
    my $exclude_glob = '{' .
        join(',', map { s/,/\\,/rg } @{$opts{exclude_globs}}) .
    '}';
    my $filter_ignore = sub {
        # Filter out files that are not going to be included in the debian
        # tarball due to ignores.
        my %exclude;
        my $reldir = File::Spec->abs2rel($File::Find::dir, $self->{dir});
        my $cwd = getcwd();
        # Apply the pattern both from the top dir and from the inspected dir
        chdir $self->{dir}
            or syserr(g_("unable to chdir to '%s'"), $self->{dir});
        $exclude{$_} = 1 foreach glob $exclude_glob;
        chdir $cwd or syserr(g_("unable to chdir to '%s'"), $cwd);
        chdir $File::Find::dir
            or syserr(g_("unable to chdir to '%s'"), $File::Find::dir);
        $exclude{$_} = 1 foreach glob $exclude_glob;
        chdir $cwd or syserr(g_("unable to chdir to '%s'"), $cwd);
        my @result;
        foreach my $fn (@_) {
            unless (exists $exclude{$fn} or exists $exclude{"$reldir/$fn"}) {
                push @result, $fn;
            }
        }
        return @result;
    };
    find({ wanted => $check_binary, preprocess => $filter_ignore,
           no_chdir => 1 }, File::Spec->catdir($self->{dir}, 'debian'));
    error(P_('detected %d unwanted binary file (add it in ' .
             'debian/source/include-binaries to allow its inclusion).',
             'detected %d unwanted binary files (add them in ' .
             'debian/source/include-binaries to allow their inclusion).',
             $unwanted_binaries), $unwanted_binaries)
        if $unwanted_binaries;
}

1;
