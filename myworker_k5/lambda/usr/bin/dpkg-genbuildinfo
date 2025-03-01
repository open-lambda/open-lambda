#!/usr/bin/perl
#
# dpkg-genbuildinfo
#
# Copyright © 1996 Ian Jackson
# Copyright © 2000,2001 Wichert Akkerman
# Copyright © 2003-2013 Yann Dirson <dirson@debian.org>
# Copyright © 2006-2016 Guillem Jover <guillem@debian.org>
# Copyright © 2014 Niko Tyni <ntyni@debian.org>
# Copyright © 2014-2015 Jérémy Bobbio <lunar@debian.org>
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

use strict;
use warnings;

use List::Util qw(any);
use Cwd;
use File::Basename;
use POSIX qw(:fcntl_h :locale_h strftime);

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::Checksums;
use Dpkg::ErrorHandling;
use Dpkg::Arch qw(get_build_arch get_host_arch debarch_eq);
use Dpkg::Build::Types;
use Dpkg::Build::Info qw(get_build_env_allowed);
use Dpkg::BuildOptions;
use Dpkg::BuildFlags;
use Dpkg::BuildProfiles qw(get_build_profiles);
use Dpkg::Control::Info;
use Dpkg::Control::Fields;
use Dpkg::Control;
use Dpkg::Changelog::Parse;
use Dpkg::Deps;
use Dpkg::Dist::Files;
use Dpkg::Lock;
use Dpkg::Version;
use Dpkg::Vendor qw(get_current_vendor run_vendor_hook);

textdomain('dpkg-dev');

my $controlfile = 'debian/control';
my $changelogfile = 'debian/changelog';
my $changelogformat;
my $fileslistfile = 'debian/files';
my $uploadfilesdir = '..';
my $outputfile;
my $stdout = 0;
my $admindir = $Dpkg::ADMINDIR;
my %use_feature = (
    kernel => 0,
    path => 0,
);
my @build_profiles = get_build_profiles();
my $buildinfo_format = '1.0';
my $buildinfo;

my $checksums = Dpkg::Checksums->new();
my %distbinaries;
my %archadded;
my @archvalues;

sub get_build_date {
    my $date;

    setlocale(LC_TIME, 'C');
    $date = strftime('%a, %d %b %Y %T %z', localtime);
    setlocale(LC_TIME, '');

    return $date;
}

# There is almost the same function in dpkg-checkbuilddeps, they probably
# should be factored out.
sub parse_status {
    my $status = shift;

    my $facts = Dpkg::Deps::KnownFacts->new();
    my %depends;
    my @essential_pkgs;

    local $/ = '';
    open my $status_fh, '<', $status or syserr(g_('cannot open %s'), $status);
    while (<$status_fh>) {
        next unless /^Status: .*ok installed$/m;

        my ($package) = /^Package: (.*)$/m;
        my ($version) = /^Version: (.*)$/m;
        my ($arch) = /^Architecture: (.*)$/m;
        my ($multiarch) = /^Multi-Arch: (.*)$/m;

        $facts->add_installed_package($package, $version, $arch, $multiarch);

        if (/^Essential: yes$/m) {
            push @essential_pkgs, $package;
        }

        if (/^Provides: (.*)$/m) {
            my $provides = deps_parse($1, reduce_arch => 1, union => 1);

            next if not defined $provides;

            deps_iterate($provides, sub {
                my $dep = shift;
                $facts->add_provided_package($dep->{package}, $dep->{relation},
                                             $dep->{version}, $package);
            });
        }

        foreach my $deptype (qw(Pre-Depends Depends)) {
            next unless /^$deptype: (.*)$/m;

            my $depends = $1;
            foreach (split /,\s*/, $depends) {
                push @{$depends{"$package:$arch"}}, $_;
            }
        }
    }
    close $status_fh;

    return ($facts, \%depends, \@essential_pkgs);
}

sub append_deps {
    my $pkgs = shift;

    foreach my $dep_str (@_) {
        next unless $dep_str;

        my $deps = deps_parse($dep_str, reduce_restrictions => 1,
                              build_dep => 1,
                              build_profiles => \@build_profiles);

        # We add every sub-dependencies as we cannot know which package in
        # an OR dependency has been effectively used.
        deps_iterate($deps, sub {
            push @{$pkgs},
                $_[0]->{package} . (defined $_[0]->{archqual} ? ':' . $_[0]->{archqual} : '');
            1
        });
    }
}

sub collect_installed_builddeps {
    my $control = shift;

    my ($facts, $depends, $essential_pkgs) = parse_status("$admindir/status");
    my %seen_pkgs;
    my @unprocessed_pkgs;

    # Parse essential packages list.
    append_deps(\@unprocessed_pkgs,
                @{$essential_pkgs},
                run_vendor_hook('builtin-build-depends'),
                $control->get_source->{'Build-Depends'});

    if (build_has_any(BUILD_ARCH_DEP)) {
        append_deps(\@unprocessed_pkgs,
                    $control->get_source->{'Build-Depends-Arch'});
    }

    if (build_has_any(BUILD_ARCH_INDEP)) {
        append_deps(\@unprocessed_pkgs,
                    $control->get_source->{'Build-Depends-Indep'});
    }

    my $installed_deps = Dpkg::Deps::AND->new();

    while (my $pkg_name = shift @unprocessed_pkgs) {
        next if $seen_pkgs{$pkg_name};
        $seen_pkgs{$pkg_name} = 1;

        my $required_architecture;
        if ($pkg_name =~ /\A(.*):(.*)\z/) {
            $pkg_name = $1;
            my $arch = $2;
            $required_architecture = $arch if $arch !~ /\A(?:all|any|native)\Z/
        }
        my $pkg;
        my $qualified_pkg_name;
        foreach my $installed_pkg (@{$facts->{pkg}->{$pkg_name}}) {
            if (!defined $required_architecture ||
                $required_architecture eq $installed_pkg->{architecture}) {
                $pkg = $installed_pkg;
                $qualified_pkg_name = $pkg_name . ':' . $installed_pkg->{architecture};
                last;
            }
        }
        if (defined $pkg) {
            my $version = $pkg->{version};
            my $architecture = $pkg->{architecture};
            my $new_deps_str = defined $depends->{$qualified_pkg_name} ? deps_concat(@{$depends->{$qualified_pkg_name}}) : '';
            my $new_deps = deps_parse($new_deps_str);
            if (!defined $required_architecture) {
                $installed_deps->add(Dpkg::Deps::Simple->new("$pkg_name (= $version)"));
            } else {
                $installed_deps->add(Dpkg::Deps::Simple->new("$qualified_pkg_name (= $version)"));

                # Dependencies of foreign packages are also foreign packages
                # (or Arch:all) so we need to qualify them as well. We figure
                # out if the package is actually foreign by searching for an
                # installed package of the right architecture.
                deps_iterate($new_deps, sub {
                    my $dep = shift;
                    return unless defined $facts->{pkg}->{$dep->{package}};
                    $dep->{archqual} //= $architecture
                        if any { $_[0]->{architecture} eq $architecture }, @{$facts->{pkg}->{$dep->{package}}};
                    1;
                });
            }

            # We add every sub-dependencies as we cannot know which package
            # in an OR dependency has been effectively used.
            deps_iterate($new_deps, sub {
                push @unprocessed_pkgs,
                     $_[0]->{package} . (defined $_[0]->{archqual} ? ':' . $_[0]->{archqual} : '');
                1
            });
        } elsif (defined $facts->{virtualpkg}->{$pkg_name}) {
            # virtual package: we cannot know for sure which implementation
            # is the one that has been used, so let's add them all...
            foreach my $provided (@{$facts->{virtualpkg}->{$pkg_name}}) {
                push @unprocessed_pkgs, $provided->{provider};
            }
        }
        # else: it is a package in an OR dependency that has been otherwise
        # satisfied.
    }
    $installed_deps->simplify_deps(Dpkg::Deps::KnownFacts->new());
    $installed_deps->sort();
    $installed_deps = "\n" . $installed_deps->output();
    $installed_deps =~ s/, /,\n/g;

    return $installed_deps;
}

sub cleansed_environment {
    # Consider only allowed variables which are not supposed to leak
    # local user information.
    my %env = map {
        $_ => $ENV{$_}
    } grep {
        exists $ENV{$_}
    } get_build_env_allowed();

    # Record flags from dpkg-buildflags.
    my $bf = Dpkg::BuildFlags->new();
    $bf->load_system_config();
    $bf->load_user_config();
    $bf->load_environment_config();
    foreach my $flag ($bf->list()) {
        next if $bf->get_origin($flag) eq 'vendor';

        # We do not need to record *_{STRIP,APPEND,PREPEND} as they
        # have been used already to compute the above value.
        $env{"DEB_${flag}_SET"} = $bf->get($flag);
    }

    return join "\n", map { $_ . '="' . ($env{$_} =~ s/"/\\"/gr) . '"' }
                      sort keys %env;
}

sub version {
    printf g_("Debian %s version %s.\n"), $Dpkg::PROGNAME, $Dpkg::PROGVERSION;

    printf g_('
This is free software; see the GNU General Public License version 2 or
later for copying conditions. There is NO warranty.
');
}

sub usage {
    printf g_(
'Usage: %s [<option>...]')
    . "\n\n" . g_(
"Options:
  --build=<type>[,...]     specify the build <type>: full, source, binary,
                             any, all (default is \'full\').
  -c<control-file>         get control info from this file.
  -l<changelog-file>       get per-version info from this file.
  -f<files-list-file>      get .deb files list from this file.
  -F<changelog-format>     force changelog format.
  -O[<buildinfo-file>]     write to stdout (or <buildinfo-file>).
  -u<upload-files-dir>     directory with files (default is '..').
  --always-include-kernel  always include Build-Kernel-Version.
  --always-include-path    always include Build-Path.
  --admindir=<directory>   change the administrative directory.
  -?, --help               show this help message.
      --version            show the version.
"), $Dpkg::PROGNAME;
}

my $build_opts = Dpkg::BuildOptions->new();
$build_opts->parse_features('buildinfo', \%use_feature);

while (@ARGV) {
    $_ = shift @ARGV ;
    if (m/^--build=(.*)$/) {
        set_build_type_from_options($1, $_);
    } elsif (m/^-c(.*)$/) {
        $controlfile = $1;
    } elsif (m/^-l(.*)$/) {
        $changelogfile = $1;
    } elsif (m/^-f(.*)$/) {
        $fileslistfile = $1;
    } elsif (m/^-F([0-9a-z]+)$/) {
        $changelogformat = $1;
    } elsif (m/^-u(.*)$/) {
        $uploadfilesdir = $1;
    } elsif (m/^-O$/) {
        $stdout = 1;
    } elsif (m/^-O(.*)$/) {
        $outputfile = $1;
    } elsif (m/^--buildinfo-id=.*$/) {
        # Deprecated option
        warning('--buildinfo-id is deprecated, it is without effect');
    } elsif (m/^--always-include-kernel$/) {
        $use_feature{kernel} = 1;
    } elsif (m/^--always-include-path$/) {
        $use_feature{path} = 1;
    } elsif (m/^--admindir=(.*)$/) {
        $admindir = $1;
    } elsif (m/^-(?:\?|-help)$/) {
        usage();
        exit(0);
    } elsif (m/^--version$/) {
        version();
        exit(0);
    } else {
        usageerr(g_("unknown option '%s'"), $_);
    }
}

my $control = Dpkg::Control::Info->new($controlfile);
my $fields = Dpkg::Control->new(type => CTRL_FILE_BUILDINFO);
my $dist = Dpkg::Dist::Files->new();

# Retrieve info from the current changelog entry.
my %options = (file => $changelogfile);
$options{changelogformat} = $changelogformat if $changelogformat;
my $changelog = changelog_parse(%options);

# Retrieve info from the former changelog entry to handle binNMUs.
$options{count} = 1;
$options{offset} = 1;
my $prev_changelog = changelog_parse(%options);

my $sourceversion = $changelog->{'Binary-Only'} ?
                    $prev_changelog->{'Version'} : $changelog->{'Version'};
my $binaryversion = Dpkg::Version->new($changelog->{'Version'});

# Include .dsc if available.
my $spackage = $changelog->{'Source'};
(my $sversion = $sourceversion) =~ s/^\d+://;

if (build_has_any(BUILD_SOURCE)) {
    my $dsc = "${spackage}_${sversion}.dsc";

    $checksums->add_from_file("$uploadfilesdir/$dsc", key => $dsc);

    push @archvalues, 'source';
}

my $dist_count = 0;

$dist_count = $dist->load($fileslistfile) if -e $fileslistfile;

if (build_has_any(BUILD_BINARY)) {
    error(g_('binary build with no binary artifacts found; .buildinfo is meaningless'))
        if $dist_count == 0;

    foreach my $file ($dist->get_files()) {
        # Make us a bit idempotent.
        next if $file->{filename} =~ m/\.buildinfo$/;

        if (defined $file->{arch}) {
            my $arch_all = debarch_eq('all', $file->{arch});

            next if build_has_none(BUILD_ARCH_INDEP) and $arch_all;
            next if build_has_none(BUILD_ARCH_DEP) and not $arch_all;

            $distbinaries{$file->{package}} = 1 if defined $file->{package};
        }

        my $path = "$uploadfilesdir/$file->{filename}";
        $checksums->add_from_file($path, key => $file->{filename});

        if (defined $file->{package_type} and $file->{package_type} =~ m/^u?deb$/) {
            push @archvalues, $file->{arch}
                if defined $file->{arch} and not $archadded{$file->{arch}}++;
        }
    }
}

$fields->{'Format'} = $buildinfo_format;
$fields->{'Source'} = $spackage;
$fields->{'Binary'} = join(' ', sort keys %distbinaries);
# Avoid overly long line by splitting over multiple lines.
if (length($fields->{'Binary'}) > 980) {
    $fields->{'Binary'} =~ s/(.{0,980}) /$1\n/g;
}

$fields->{'Architecture'} = join ' ', sort @archvalues;
$fields->{'Version'} = $binaryversion;

if ($changelog->{'Binary-Only'}) {
    $fields->{'Source'} .= ' (' . $sourceversion . ')';
    $fields->{'Binary-Only-Changes'} =
        $changelog->{'Changes'} . "\n\n"
        . ' -- ' . $changelog->{'Maintainer'}
        . '  ' . $changelog->{'Date'};
}

$fields->{'Build-Origin'} = get_current_vendor();
$fields->{'Build-Architecture'} = get_build_arch();
$fields->{'Build-Date'} = get_build_date();

if ($use_feature{kernel}) {
    my (undef, undef, $kern_rel, $kern_ver, undef) = POSIX::uname();
    $fields->{'Build-Kernel-Version'} = "$kern_rel $kern_ver";
}

my $cwd = getcwd();
if ($use_feature{path}) {
    $fields->{'Build-Path'} = $cwd;
} else {
    # Only include the build path if its root path is considered acceptable
    # by the vendor.
    foreach my $root_path (run_vendor_hook('builtin-system-build-paths')) {
        if (index($cwd, $root_path) == 0) {
            $fields->{'Build-Path'} = $cwd;
            last;
        }
    }
}

$fields->{'Build-Tainted-By'} = "\n" . join "\n", run_vendor_hook('build-tainted-by');

$checksums->export_to_control($fields);

$fields->{'Installed-Build-Depends'} = collect_installed_builddeps($control);

$fields->{'Environment'} = "\n" . cleansed_environment();

# Generate the buildinfo filename.
if ($stdout) {
    # Nothing to do.
} elsif (defined $outputfile) {
    $buildinfo = basename($outputfile);
} else {
    my $arch;

    if (build_has_any(BUILD_ARCH_DEP)) {
        $arch = get_host_arch();
    } elsif (build_has_any(BUILD_ARCH_INDEP)) {
        $arch = 'all';
    } elsif (build_has_any(BUILD_SOURCE)) {
        $arch = 'source';
    }

    my $bversion = $binaryversion->as_string(omit_epoch => 1);
    $buildinfo = "${spackage}_${bversion}_${arch}.buildinfo";
    $outputfile = "$uploadfilesdir/$buildinfo";
}

# Write out the generated .buildinfo file.

if ($stdout) {
    $fields->output(\*STDOUT);
} else {
    my $section = $control->get_source->{'Section'} || '-';
    my $priority = $control->get_source->{'Priority'} || '-';

    # Obtain a lock on debian/control to avoid simultaneous updates
    # of debian/files when parallel building is in use
    my $lockfh;
    my $lockfile = 'debian/control';
    $lockfile = $controlfile if not -e $lockfile;

    sysopen $lockfh, $lockfile, O_WRONLY
        or syserr(g_('cannot write %s'), $lockfile);
    file_lock($lockfh, $lockfile);

    $dist = Dpkg::Dist::Files->new();
    $dist->load($fileslistfile) if -e $fileslistfile;

    foreach my $file ($dist->get_files()) {
        if (defined $file->{package} &&
            $file->{package} eq $spackage &&
            $file->{package_type} eq 'buildinfo' &&
            (debarch_eq($file->{arch}, $fields->{'Architecture'}) ||
             debarch_eq($file->{arch}, 'all') ||
             debarch_eq($file->{arch}, 'source'))) {
            $dist->del_file($file->{filename});
        }
    }

    $dist->add_file($buildinfo, $section, $priority);
    $dist->save("$fileslistfile.new");

    rename "$fileslistfile.new", $fileslistfile
        or syserr(g_('install new files list file'));

    # Release the lock
    close $lockfh or syserr(g_('cannot close %s'), $lockfile);

    $fields->save("$outputfile.new");

    rename "$outputfile.new", $outputfile
        or syserr(g_("cannot install output buildinfo file '%s'"), $outputfile);
}

1;
