#!/usr/bin/perl
#
# dpkg-genchanges
#
# Copyright © 1996 Ian Jackson
# Copyright © 2000,2001 Wichert Akkerman
# Copyright © 2006-2014 Guillem Jover <guillem@debian.org>
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

use List::Util qw(any all none);
use Encode;
use POSIX qw(:errno_h :locale_h);

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::File;
use Dpkg::Checksums;
use Dpkg::ErrorHandling;
use Dpkg::Build::Types;
use Dpkg::BuildProfiles qw(get_build_profiles parse_build_profiles
                           evaluate_restriction_formula);
use Dpkg::Arch qw(get_host_arch debarch_eq debarch_is debarch_list_parse);
use Dpkg::Compression;
use Dpkg::Control::Info;
use Dpkg::Control::Fields;
use Dpkg::Control;
use Dpkg::Substvars;
use Dpkg::Vars;
use Dpkg::Changelog::Parse;
use Dpkg::Dist::Files;
use Dpkg::Version;

textdomain('dpkg-dev');

my $controlfile = 'debian/control';
my $changelogfile = 'debian/changelog';
my $changelogformat;
my $fileslistfile = 'debian/files';
my $outputfile;
my $uploadfilesdir = '..';
my $sourcestyle = 'i';
my $quiet = 0;
my $host_arch = get_host_arch();
my @profiles = get_build_profiles();
my $changes_format = '1.8';

my %p2f;           # - package to file map, has entries for "packagename"
my %f2seccf;       # - package to section map, from control file
my %f2pricf;       # - package to priority map, from control file
my %sourcedefault; # - default values as taken from source (used for Section,
                   #   Priority and Maintainer)

my @descriptions;

my $checksums = Dpkg::Checksums->new();
my %remove;        # - fields to remove
my %override;
my %archadded;
my @archvalues;
my $changesdescription;
my $forcemaint;
my $forcechangedby;
my $since;

my $substvars_loaded = 0;
my $substvars = Dpkg::Substvars->new();
$substvars->set_as_auto('Format', $changes_format);

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
  -g                       source and arch-indep build.
  -G                       source and arch-specific build.
  -b                       binary-only, no source files.
  -B                       binary-only, only arch-specific files.
  -A                       binary-only, only arch-indep files.
  -S                       source-only, no binary files.
  -c<control-file>         get control info from this file.
  -l<changelog-file>       get per-version info from this file.
  -f<files-list-file>      get .deb files list from this file.
  -v<since-version>        include all changes later than version.
  -C<changes-description>  use change description from this file.
  -m<maintainer>           override control's maintainer value.
  -e<maintainer>           override changelog's maintainer value.
  -u<upload-files-dir>     directory with files (default is '..').
  -si                      source includes orig, if new upstream (default).
  -sa                      source includes orig, always.
  -sd                      source is diff and .dsc only.
  -q                       quiet - no informational messages on stderr.
  -F<changelog-format>     force changelog format.
  -V<name>=<value>         set a substitution variable.
  -T<substvars-file>       read variables here, not debian/substvars.
  -D<field>=<value>        override or add a field and value.
  -U<field>                remove a field.
  -O[<filename>]           write to stdout (default) or <filename>.
  -?, --help               show this help message.
      --version            show the version.
"), $Dpkg::PROGNAME;
}


while (@ARGV) {
    $_=shift(@ARGV);
    if (m/^--build=(.*)$/) {
        set_build_type_from_options($1, $_);
    } elsif (m/^-b$/) {
	set_build_type(BUILD_BINARY, $_);
    } elsif (m/^-B$/) {
	set_build_type(BUILD_ARCH_DEP, $_);
    } elsif (m/^-A$/) {
	set_build_type(BUILD_ARCH_INDEP, $_);
    } elsif (m/^-S$/) {
	set_build_type(BUILD_SOURCE, $_);
    } elsif (m/^-G$/) {
	set_build_type(BUILD_SOURCE | BUILD_ARCH_DEP, $_);
    } elsif (m/^-g$/) {
	set_build_type(BUILD_SOURCE | BUILD_ARCH_INDEP, $_);
    } elsif (m/^-s([iad])$/) {
        $sourcestyle= $1;
    } elsif (m/^-q$/) {
        $quiet= 1;
    } elsif (m/^-c(.*)$/) {
	$controlfile = $1;
    } elsif (m/^-l(.*)$/) {
	$changelogfile = $1;
    } elsif (m/^-C(.*)$/) {
	$changesdescription = $1;
    } elsif (m/^-f(.*)$/) {
	$fileslistfile = $1;
    } elsif (m/^-v(.*)$/) {
	$since = $1;
    } elsif (m/^-T(.*)$/) {
	$substvars->load($1) if -e $1;
	$substvars_loaded = 1;
    } elsif (m/^-m(.*)$/s) {
	$forcemaint = $1;
    } elsif (m/^-e(.*)$/s) {
	$forcechangedby = $1;
    } elsif (m/^-F([0-9a-z]+)$/) {
        $changelogformat = $1;
    } elsif (m/^-D([^\=:]+)[=:](.*)$/s) {
	$override{$1} = $2;
    } elsif (m/^-u(.*)$/) {
	$uploadfilesdir = $1;
    } elsif (m/^-U([^\=:]+)$/) {
        $remove{$1} = 1;
    } elsif (m/^-V(\w[-:0-9A-Za-z]*)[=:](.*)$/s) {
	$substvars->set($1, $2);
    } elsif (m/^-O(.*)$/) {
        $outputfile = $1;
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

# Do not pollute STDOUT with info messages if the .changes file goes there.
if (not defined $outputfile) {
    report_options(info_fh => \*STDERR, quiet_warnings => $quiet);
    $outputfile = '-';
}

# Retrieve info from the current changelog entry
my %options = (file => $changelogfile);
$options{changelogformat} = $changelogformat if $changelogformat;
$options{since} = $since if defined($since);
my $changelog = changelog_parse(%options);
# Change options to retrieve info of the former changelog entry
delete $options{since};
$options{count} = 1;
$options{offset} = 1;
my $prev_changelog = changelog_parse(%options);
# Other initializations
my $control = Dpkg::Control::Info->new($controlfile);
my $fields = Dpkg::Control->new(type => CTRL_FILE_CHANGES);

my $sourceversion = $changelog->{'Binary-Only'} ?
                    $prev_changelog->{'Version'} : $changelog->{'Version'};
my $binaryversion = $changelog->{'Version'};

$substvars->set_version_substvars($sourceversion, $binaryversion);
$substvars->set_vendor_substvars();
$substvars->set_arch_substvars();
$substvars->load('debian/substvars') if -e 'debian/substvars' and not $substvars_loaded;

if (defined($prev_changelog) and
    version_compare_relation($changelog->{'Version'}, REL_LT,
                             $prev_changelog->{'Version'}))
{
    warning(g_('the current version (%s) is earlier than the previous one (%s)'),
	$changelog->{'Version'}, $prev_changelog->{'Version'})
        # Backports have lower version number by definition.
        unless $changelog->{'Version'} =~ /~(?:bpo|deb)/;
}

# Scan control info of source package
my $src_fields = $control->get_source();
foreach (keys %{$src_fields}) {
    my $v = $src_fields->{$_};
    if (m/^Source$/) {
        set_source_package($v);
    } elsif (m/^Section$|^Priority$/i) {
        $sourcedefault{$_} = $v;
    } elsif (m/^Description$/i) {
        # Description in changes is computed, do not copy this field, only
        # initialize the description substvars.
        $substvars->set_desc_substvars($v);
    } else {
        field_transfer_single($src_fields, $fields);
    }
}

my $dist = Dpkg::Dist::Files->new();
my $origsrcmsg;

if (build_has_any(BUILD_SOURCE)) {
    my $sec = $sourcedefault{'Section'} // '-';
    my $pri = $sourcedefault{'Priority'} // '-';
    warning(g_('missing Section for source files')) if $sec eq '-';
    warning(g_('missing Priority for source files')) if $pri eq '-';

    my $spackage = get_source_package();
    (my $sversion = $substvars->get('source:Version')) =~ s/^\d+://;

    my $dsc = "${spackage}_${sversion}.dsc";
    my $dsc_pathname = "$uploadfilesdir/$dsc";
    my $dsc_fields = Dpkg::Control->new(type => CTRL_PKG_SRC);
    $dsc_fields->load($dsc_pathname) or error(g_('%s is empty'), $dsc_pathname);
    $checksums->add_from_file($dsc_pathname, key => $dsc);
    $checksums->add_from_control($dsc_fields, use_files_for_md5 => 1);

    # Compare upstream version to previous upstream version to decide if
    # the .orig tarballs must be included
    my $include_tarball;
    if (defined($prev_changelog)) {
        my $cur = Dpkg::Version->new($changelog->{'Version'});
        my $prev = Dpkg::Version->new($prev_changelog->{'Version'});
        if ($cur->version() ne $prev->version()) {
            $include_tarball = 1;
        } elsif ($changelog->{'Source'} ne $prev_changelog->{'Source'}) {
            $include_tarball = 1;
        } else {
            $include_tarball = 0;
        }
    } else {
        # No previous entry means first upload, tarball required
        $include_tarball = 1;
    }

    my $ext = compression_get_file_extension_regex();
    if ((($sourcestyle =~ m/i/ && !$include_tarball) ||
         $sourcestyle =~ m/d/) &&
        any { m/\.(?:debian\.tar|diff)\.$ext$/ } $checksums->get_files())
    {
        $origsrcmsg = g_('not including original source code in upload');
        foreach my $f (grep { m/\.orig(-.+)?\.tar\.$ext$/ } $checksums->get_files()) {
            $checksums->remove_file($f);
            $checksums->remove_file("$f.asc");
        }
    } else {
        if ($sourcestyle =~ m/d/ &&
            none { m/\.(?:debian\.tar|diff)\.$ext$/ } $checksums->get_files()) {
            warning(g_('ignoring -sd option for native Debian package'));
        }
        $origsrcmsg = g_('including full source code in upload');
    }

    push @archvalues, 'source';

    # Only add attributes for files being distributed.
    for my $f ($checksums->get_files()) {
        $dist->add_file($f, $sec, $pri);
    }
} elsif (build_is(BUILD_ARCH_DEP)) {
    $origsrcmsg = g_('binary-only arch-specific upload ' .
                     '(source code and arch-indep packages not included)');
} elsif (build_is(BUILD_ARCH_INDEP)) {
    $origsrcmsg = g_('binary-only arch-indep upload ' .
                     '(source code and arch-specific packages not included)');
} else {
    $origsrcmsg = g_('binary-only upload (no source code included)');
}

my $dist_binaries = 0;

$dist->load($fileslistfile) if -e $fileslistfile;

foreach my $file ($dist->get_files()) {
    my $f = $file->{filename};

    if (defined $file->{package} && $file->{package_type} eq 'buildinfo') {
        # We always distribute the .buildinfo file.
        $checksums->add_from_file("$uploadfilesdir/$f", key => $f);
        next;
    }

    # If this is a source-only upload, ignore any other artifacts.
    next if build_has_none(BUILD_BINARY);

    if (defined $file->{arch}) {
        my $arch_all = debarch_eq('all', $file->{arch});

        next if build_has_none(BUILD_ARCH_INDEP) and $arch_all;
        next if build_has_none(BUILD_ARCH_DEP) and not $arch_all;

        push @archvalues, $file->{arch} if not $archadded{$file->{arch}}++;
    }
    if (defined $file->{package} && $file->{package_type} =~ m/^u?deb$/) {
        $p2f{$file->{package}} //= [];
        push @{$p2f{$file->{package}}}, $file->{filename};
    }

    $checksums->add_from_file("$uploadfilesdir/$f", key => $f);
    $dist_binaries++;
}

error(g_('binary build with no binary artifacts found; cannot distribute'))
    if build_has_any(BUILD_BINARY) && $dist_binaries == 0;

# Scan control info of all binary packages
foreach my $pkg ($control->get_packages()) {
    my $p = $pkg->{'Package'};
    my $a = $pkg->{'Architecture'};
    my $bp = $pkg->{'Build-Profiles'};
    my $d = $pkg->{'Description'} || 'no description available';
    $d = $1 if $d =~ /^(.*)\n/;
    my $pkg_type = $pkg->{'Package-Type'} ||
                   $pkg->get_custom_field('Package-Type') || 'deb';

    my @restrictions;
    @restrictions = parse_build_profiles($bp) if defined $bp;

    if (not defined($p2f{$p})) {
	# No files for this package... warn if it's unexpected
	if (((build_has_any(BUILD_ARCH_INDEP) and debarch_eq('all', $a)) or
	     (build_has_any(BUILD_ARCH_DEP) and
	      (any { debarch_is($host_arch, $_) } debarch_list_parse($a, positive => 1)))) and
	    (@restrictions == 0 or
	     evaluate_restriction_formula(\@restrictions, \@profiles)))
	{
	    warning(g_('package %s in control file but not in files list'),
		    $p);
	}
	next; # and skip it
    }

    # Add description of all binary packages
    $d = $substvars->substvars($d);
    my $desc = encode_utf8(sprintf('%-10s - %-.65s', $p, decode_utf8($d)));
    $desc .= " ($pkg_type)" if $pkg_type ne 'deb';
    push @descriptions, $desc;

    # List of files for this binary package.
    my @f = @{$p2f{$p}};

    foreach (keys %{$pkg}) {
	my $v = $pkg->{$_};

	if (m/^Section$/) {
	    $f2seccf{$_} = $v foreach (@f);
	} elsif (m/^Priority$/) {
	    $f2pricf{$_} = $v foreach (@f);
	} elsif (m/^Architecture$/) {
	    if (build_has_any(BUILD_ARCH_DEP) and
	        (any { debarch_is($host_arch, $_) } debarch_list_parse($v, positive => 1))) {
		$v = $host_arch;
	    } elsif (!debarch_eq('all', $v)) {
		$v = '';
	    }
	    push(@archvalues, $v) if $v and not $archadded{$v}++;
        } elsif (m/^Description$/) {
            # Description in changes is computed, do not copy this field
	} else {
            field_transfer_single($pkg, $fields);
	}
    }
}

# Scan fields of dpkg-parsechangelog
foreach (keys %{$changelog}) {
    my $v = $changelog->{$_};
    if (m/^Source$/i) {
	set_source_package($v);
    } elsif (m/^Maintainer$/i) {
	$fields->{'Changed-By'} = $v;
    } else {
        field_transfer_single($changelog, $fields);
    }
}

if ($changesdescription) {
    $fields->{'Changes'} = "\n" . file_slurp($changesdescription);
}

for my $p (keys %p2f) {
    if (not defined $control->get_pkg_by_name($p)) {
        # Skip automatically generated packages (such as debugging symbol
        # packages), by using the Auto-Built-Package field.
        next if all {
            my $file = $dist->get_file($_);

            $file->{attrs}->{automatic} eq 'yes'
        } @{$p2f{$p}};

        warning(g_('package %s listed in files list but not in control info'), $p);
        next;
    }

    foreach my $f (@{$p2f{$p}}) {
	my $file = $dist->get_file($f);

	my $sec = $f2seccf{$f} || $sourcedefault{'Section'} // '-';
	if ($sec eq '-') {
	    warning(g_("missing Section for binary package %s; using '-'"), $p);
	}
	if ($sec ne $file->{section}) {
	    error(g_('package %s has section %s in control file but %s in ' .
	             'files list'), $p, $sec, $file->{section});
	}

	my $pri = $f2pricf{$f} || $sourcedefault{'Priority'} // '-';
	if ($pri eq '-') {
	    warning(g_("missing Priority for binary package %s; using '-'"), $p);
	}
	if ($pri ne $file->{priority}) {
	    error(g_('package %s has priority %s in control file but %s in ' .
	             'files list'), $p, $pri, $file->{priority});
	}
    }
}

info($origsrcmsg);

$fields->{'Format'} = $substvars->get('Format');

if (length $fields->{'Date'} == 0) {
    setlocale(LC_TIME, 'C');
    $fields->{'Date'} = POSIX::strftime('%a, %d %b %Y %T %z', localtime);
    setlocale(LC_TIME, '');
}

$fields->{'Binary'} = join ' ', sort keys %p2f;
# Avoid overly long line by splitting over multiple lines
if (length($fields->{'Binary'}) > 980) {
    $fields->{'Binary'} =~ s/(.{0,980}) /$1\n/g;
}

$fields->{'Architecture'} = join ' ', @archvalues;

$fields->{'Built-For-Profiles'} = join ' ', get_build_profiles();

$fields->{'Description'} = "\n" . join("\n", sort @descriptions);

$fields->{'Files'} = '';

foreach my $f ($checksums->get_files()) {
    my $file = $dist->get_file($f);

    $fields->{'Files'} .= "\n" . $checksums->get_checksum($f, 'md5') .
			  ' ' . $checksums->get_size($f) .
			  " $file->{section} $file->{priority} $f";
}
$checksums->export_to_control($fields);
# redundant with the Files field
delete $fields->{'Checksums-Md5'};

$fields->{'Source'} = get_source_package();
if ($fields->{'Version'} ne $substvars->get('source:Version')) {
    $fields->{'Source'} .= ' (' . $substvars->get('source:Version') . ')';
}

$fields->{'Maintainer'} = $forcemaint if defined($forcemaint);
$fields->{'Changed-By'} = $forcechangedby if defined($forcechangedby);

for my $f (qw(Version Distribution Maintainer Changes)) {
    error(g_('missing information for critical output field %s'), $f)
        unless defined $fields->{$f};
}

for my $f (qw(Urgency)) {
    warning(g_('missing information for output field %s'), $f)
        unless defined $fields->{$f};
}

for my $f (keys %override) {
    $fields->{$f} = $override{$f};
}
for my $f (keys %remove) {
    delete $fields->{$f};
}

# Note: do not perform substitution of variables, one of the reasons is that
# they could interfere with field values, for example the Changes field.
$fields->save($outputfile);
