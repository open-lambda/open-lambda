#!/usr/bin/perl
#
# dpkg-gencontrol
#
# Copyright © 1996 Ian Jackson
# Copyright © 2000,2002 Wichert Akkerman
# Copyright © 2006-2015 Guillem Jover <guillem@debian.org>
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

use List::Util qw(none);
use POSIX qw(:errno_h :fcntl_h);
use File::Find;

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Lock;
use Dpkg::Arch qw(get_host_arch debarch_eq debarch_is debarch_list_parse);
use Dpkg::Package;
use Dpkg::BuildProfiles qw(get_build_profiles);
use Dpkg::Deps;
use Dpkg::Control;
use Dpkg::Control::Info;
use Dpkg::Control::Fields;
use Dpkg::Substvars;
use Dpkg::Vars;
use Dpkg::Changelog::Parse;
use Dpkg::Dist::Files;

textdomain('dpkg-dev');


my $controlfile = 'debian/control';
my $changelogfile = 'debian/changelog';
my $changelogformat;
my $fileslistfile = 'debian/files';
my $packagebuilddir = 'debian/tmp';
my $outputfile;

my $sourceversion;
my $binaryversion;
my $forceversion;
my $forcefilename;
my $stdout;
my %remove;
my %override;
my $oppackage;
my $substvars = Dpkg::Substvars->new();
my $substvars_loaded = 0;


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
'Options:
  -p<package>              print control file for package.
  -c<control-file>         get control info from this file.
  -l<changelog-file>       get per-version info from this file.
  -F<changelog-format>     force changelog format.
  -v<force-version>        set version of binary package.
  -f<files-list-file>      write files here instead of debian/files.
  -P<package-build-dir>    temporary build directory instead of debian/tmp.
  -n<filename>             assume the package filename will be <filename>.
  -O[<file>]               write to stdout (or <file>), not .../DEBIAN/control.
  -is, -ip, -isp, -ips     deprecated, ignored for compatibility.
  -D<field>=<value>        override or add a field and value.
  -U<field>                remove a field.
  -V<name>=<value>         set a substitution variable.
  -T<substvars-file>       read variables here, not debian/substvars.
  -?, --help               show this help message.
      --version            show the version.
'), $Dpkg::PROGNAME;
}

while (@ARGV) {
    $_=shift(@ARGV);
    if (m/^-p/p) {
        $oppackage = ${^POSTMATCH};
        my $err = pkg_name_is_illegal($oppackage);
        error(g_("illegal package name '%s': %s"), $oppackage, $err) if $err;
    } elsif (m/^-c/p) {
        $controlfile = ${^POSTMATCH};
    } elsif (m/^-l/p) {
        $changelogfile = ${^POSTMATCH};
    } elsif (m/^-P/p) {
        $packagebuilddir = ${^POSTMATCH};
    } elsif (m/^-f/p) {
        $fileslistfile = ${^POSTMATCH};
    } elsif (m/^-v(.+)$/) {
        $forceversion= $1;
    } elsif (m/^-O$/) {
        $stdout= 1;
    } elsif (m/^-O(.+)$/) {
        $outputfile = $1;
    } elsif (m/^-i([sp][sp]?)$/) {
        warning(g_('-i%s is deprecated; it is without effect'), $1);
    } elsif (m/^-F([0-9a-z]+)$/) {
        $changelogformat=$1;
    } elsif (m/^-D([^\=:]+)[=:]/p) {
        $override{$1} = ${^POSTMATCH};
    } elsif (m/^-U([^\=:]+)$/) {
        $remove{$1}= 1;
    } elsif (m/^-V(\w[-:0-9A-Za-z]*)[=:]/p) {
        $substvars->set_as_used($1, ${^POSTMATCH});
    } elsif (m/^-T(.*)$/) {
	$substvars->load($1) if -e $1;
	$substvars_loaded = 1;
    } elsif (m/^-n/p) {
        $forcefilename = ${^POSTMATCH};
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

umask 0022; # ensure sane default permissions for created files
my %options = (file => $changelogfile);
$options{changelogformat} = $changelogformat if $changelogformat;
my $changelog = changelog_parse(%options);
if ($changelog->{'Binary-Only'}) {
    $options{count} = 1;
    $options{offset} = 1;
    my $prev_changelog = changelog_parse(%options);
    $sourceversion = $prev_changelog->{'Version'};
} else {
    $sourceversion = $changelog->{'Version'};
}

if (defined $forceversion) {
    $binaryversion = $forceversion;
} else {
    $binaryversion = $changelog->{'Version'};
}

$substvars->set_version_substvars($sourceversion, $binaryversion);
$substvars->set_vendor_substvars();
$substvars->set_arch_substvars();
$substvars->load('debian/substvars') if -e 'debian/substvars' and not $substvars_loaded;
my $control = Dpkg::Control::Info->new($controlfile);
my $fields = Dpkg::Control->new(type => CTRL_PKG_DEB);

# Old-style bin-nmus change the source version submitted to
# set_version_substvars()
$sourceversion = $substvars->get('source:Version');

my $pkg;

if (defined($oppackage)) {
    $pkg = $control->get_pkg_by_name($oppackage);
    if (not defined $pkg) {
        error(g_('package %s not in control info'), $oppackage)
    }
} else {
    my @packages = map { $_->{'Package'} } $control->get_packages();
    if (@packages == 0) {
        error(g_('no package stanza found in control info'));
    } elsif (@packages > 1) {
        error(g_('must specify package since control info has many (%s)'),
              "@packages");
    }
    $pkg = $control->get_pkg_by_idx(1);
}
$substvars->set_msg_prefix(sprintf(g_('package %s: '), $pkg->{Package}));

# Scan source package
my $src_fields = $control->get_source();
foreach (keys %{$src_fields}) {
    if (m/^Source$/) {
	set_source_package($src_fields->{$_});
    } elsif (m/^Description$/) {
        # Description in binary packages is not inherited, do not copy this
        # field, only initialize the description substvars.
        $substvars->set_desc_substvars($src_fields->{$_});
    } else {
        field_transfer_single($src_fields, $fields);
    }
}
$substvars->set_field_substvars($src_fields, 'S');

# Scan binary package
foreach (keys %{$pkg}) {
    my $v = $pkg->{$_};
    if (field_get_dep_type($_)) {
	# Delay the parsing until later
    } elsif (m/^Architecture$/) {
	my $host_arch = get_host_arch();

	if (debarch_eq('all', $v)) {
	    $fields->{$_} = $v;
	} else {
	    my @archlist = debarch_list_parse($v, positive => 1);

	    if (none { debarch_is($host_arch, $_) } @archlist) {
		error(g_("current host architecture '%s' does not " .
			 "appear in package '%s' architecture list (%s)"),
		      $host_arch, $oppackage, "@archlist");
	    }
	    $fields->{$_} = $host_arch;
	}
    } else {
        field_transfer_single($pkg, $fields);
    }
}

# Scan fields of dpkg-parsechangelog
foreach (keys %{$changelog}) {
    my $v = $changelog->{$_};

    if (m/^Source$/) {
	set_source_package($v);
    } elsif (m/^Version$/) {
        # Already handled previously.
    } elsif (m/^Maintainer$/) {
        # That field must not be copied from changelog even if it's
        # allowed in the binary package control information
    } else {
        field_transfer_single($changelog, $fields);
    }
}

$fields->{'Version'} = $binaryversion;

# Process dependency fields in a second pass, now that substvars have been
# initialized.

my $facts = Dpkg::Deps::KnownFacts->new();
$facts->add_installed_package($fields->{'Package'}, $fields->{'Version'},
                              $fields->{'Architecture'}, $fields->{'Multi-Arch'});
if (exists $pkg->{'Provides'}) {
    my $provides = deps_parse($substvars->substvars($pkg->{'Provides'}, no_warn => 1),
                              reduce_restrictions => 1, virtual => 1, union => 1);
    if (defined $provides) {
	foreach my $subdep ($provides->get_deps()) {
	    if ($subdep->isa('Dpkg::Deps::Simple')) {
		$facts->add_provided_package($subdep->{package},
                        $subdep->{relation}, $subdep->{version},
                        $fields->{'Package'});
	    }
	}
    }
}

my (@seen_deps);
foreach my $field (field_list_pkg_dep()) {
    # Arch: all can't be simplified as the host architecture is not known
    my $reduce_arch = debarch_eq('all', $pkg->{Architecture} || 'all') ? 0 : 1;
    if (exists $pkg->{$field}) {
	my $dep;
	my $field_value = $substvars->substvars($pkg->{$field},
	    msg_prefix => sprintf(g_('%s field of package %s: '), $field, $pkg->{Package}));
	if (field_get_dep_type($field) eq 'normal') {
	    $dep = deps_parse($field_value, use_arch => 1,
	                      reduce_arch => $reduce_arch,
	                      reduce_profiles => 1);
            error(g_("parsing package '%s' %s field: %s"), $oppackage,
                  $field, $field_value) unless defined $dep;
	    $dep->simplify_deps($facts, @seen_deps);
	    # Remember normal deps to simplify even further weaker deps
	    push @seen_deps, $dep;
	} else {
	    $dep = deps_parse($field_value, use_arch => 1,
	                      reduce_arch => $reduce_arch,
	                      reduce_profiles => 1, union => 1);
            error(g_("parsing package '%s' %s field: %s"), $oppackage,
                  $field, $field_value) unless defined $dep;
	    $dep->simplify_deps($facts);
            $dep->sort();
	}
	error(g_('the %s field contains an arch-specific dependency but the ' .
	         "package '%s' is architecture all"), $field, $oppackage)
	    if $dep->has_arch_restriction();
	$fields->{$field} = $dep->output();
	delete $fields->{$field} unless $fields->{$field}; # Delete empty field
    }
}

for my $f (qw(Package Version Architecture)) {
    error(g_('missing information for output field %s'), $f)
        unless defined $fields->{$f};
}
for my $f (qw(Maintainer Description)) {
    warning(g_('missing information for output field %s'), $f)
        unless defined $fields->{$f};
}

my $pkg_type = $override{'Package-Type'} ||
               $pkg->{'Package-Type'} ||
               $pkg->get_custom_field('Package-Type') || 'deb';

if ($pkg_type eq 'udeb') {
    delete $fields->{'Package-Type'};
    delete $fields->{'Homepage'};
} else {
    for my $f (qw(Subarchitecture Kernel-Version Installer-Menu-Item)) {
        warning(g_("%s package '%s' with udeb specific field %s"),
                $pkg_type, $oppackage, $f)
            if defined($fields->{$f});
    }
}

my $sourcepackage = get_source_package();
my $binarypackage = $override{'Package'} // $fields->{'Package'};
my $verdiff = $binaryversion ne $sourceversion;
if ($binarypackage ne $sourcepackage || $verdiff) {
    $fields->{'Source'} = $sourcepackage;
    $fields->{'Source'} .= ' (' . $sourceversion . ')' if $verdiff;
}

if (!defined($substvars->get('Installed-Size'))) {
    my $installed_size = 0;
    my %hardlink;
    my $scan_installed_size = sub {
        lstat or syserr(g_('cannot stat %s'), $File::Find::name);

        if (-f _ or -l _) {
            my ($dev, $ino, $nlink) = (lstat _)[0, 1, 3];

            # For filesystem objects with actual content accumulate the size
            # in 1 KiB units.
            $installed_size += POSIX::ceil((-s _) / 1024)
                if not exists $hardlink{"$dev:$ino"};

            # Track hardlinks to avoid repeated additions.
            $hardlink{"$dev:$ino"} = 1 if $nlink > 1;
        } else {
            # For other filesystem objects assume a minimum 1 KiB baseline,
            # as directories are shared resources between packages, and other
            # object types are mainly metadata-only, supposedly consuming
            # at most an inode.
            $installed_size += 1;
        }
    };
    find($scan_installed_size, $packagebuilddir) if -d $packagebuilddir;

    $substvars->set_as_auto('Installed-Size', $installed_size);
}
if (defined($substvars->get('Extra-Size'))) {
    my $size = $substvars->get('Extra-Size') + $substvars->get('Installed-Size');
    $substvars->set_as_auto('Installed-Size', $size);
}
if (defined($substvars->get('Installed-Size'))) {
    $fields->{'Installed-Size'} = $substvars->get('Installed-Size');
}

for my $f (keys %override) {
    $fields->{$f} = $override{$f};
}
for my $f (keys %remove) {
    delete $fields->{$f};
}

$fields->apply_substvars($substvars);

if ($stdout) {
    $fields->output(\*STDOUT);
} else {
    $outputfile //= "$packagebuilddir/DEBIAN/control";

    my $sversion = $fields->{'Version'};
    $sversion =~ s/^\d+://;
    $forcefilename //= sprintf('%s_%s_%s.%s', $fields->{'Package'}, $sversion,
                               $fields->{'Architecture'}, $pkg_type);
    my $section = $fields->{'Section'} || '-';
    my $priority = $fields->{'Priority'} || '-';

    # Obtain a lock on debian/control to avoid simultaneous updates
    # of debian/files when parallel building is in use
    my $lockfh;
    my $lockfile = 'debian/control';
    $lockfile = $controlfile if not -e $lockfile;

    sysopen $lockfh, $lockfile, O_WRONLY
        or syserr(g_('cannot write %s'), $lockfile);
    file_lock($lockfh, $lockfile);

    my $dist = Dpkg::Dist::Files->new();
    $dist->load($fileslistfile) if -e $fileslistfile;

    foreach my $file ($dist->get_files()) {
        if (defined $file->{package} &&
            ($file->{package} eq $fields->{'Package'}) &&
            ($file->{package_type} eq $pkg_type) &&
            (debarch_eq($file->{arch}, $fields->{'Architecture'}) ||
             debarch_eq($file->{arch}, 'all'))) {
            $dist->del_file($file->{filename});
        }
    }

    my %fileattrs;
    $fileattrs{automatic} = 'yes' if $fields->{'Auto-Built-Package'};

    $dist->add_file($forcefilename, $section, $priority, %fileattrs);
    $dist->save("$fileslistfile.new");

    rename "$fileslistfile.new", $fileslistfile
        or syserr(g_('install new files list file'));

    # Release the lock
    close $lockfh or syserr(g_('cannot close %s'), $lockfile);

    $fields->save("$outputfile.new");

    rename "$outputfile.new", $outputfile
        or syserr(g_("cannot install output control file '%s'"), $outputfile);
}

$substvars->warn_about_unused();
