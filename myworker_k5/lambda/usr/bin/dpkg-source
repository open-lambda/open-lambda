#!/usr/bin/perl
#
# dpkg-source
#
# Copyright © 1996 Ian Jackson <ijackson@chiark.greenend.org.uk>
# Copyright © 1997 Klee Dienes <klee@debian.org>
# Copyright © 1999-2003 Wichert Akkerman <wakkerma@debian.org>
# Copyright © 1999 Ben Collins <bcollins@debian.org>
# Copyright © 2000-2003 Adam Heath <doogie@debian.org>
# Copyright © 2005 Brendan O'Dea <bod@debian.org>
# Copyright © 2006-2008 Frank Lichtenheld <djpig@debian.org>
# Copyright © 2006-2009,2012 Guillem Jover <guillem@debian.org>
# Copyright © 2008-2011 Raphaël Hertzog <hertzog@debian.org>
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

use List::Util qw(any none);
use Cwd;
use File::Basename;
use File::Spec;

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Arch qw(:operators);
use Dpkg::Deps;
use Dpkg::Compression;
use Dpkg::Conf;
use Dpkg::Control::Info;
use Dpkg::Control::Tests;
use Dpkg::Control::Fields;
use Dpkg::Substvars;
use Dpkg::Version;
use Dpkg::Vars;
use Dpkg::Changelog::Parse;
use Dpkg::Source::Format;
use Dpkg::Source::Package qw(get_default_diff_ignore_regex
                             set_default_diff_ignore_regex
                             get_default_tar_ignore_pattern);
use Dpkg::Vendor qw(run_vendor_hook);

textdomain('dpkg-dev');

my $controlfile;
my $changelogfile;
my $changelogformat;

my $build_format;
my %options = (
    # Ignore files
    tar_ignore => [],
    diff_ignore_regex => '',
    # Misc options
    copy_orig_tarballs => 1,
    no_check => 0,
    no_overwrite_dir => 1,
    require_valid_signature => 0,
    require_strong_checksums => 0,
);

# Fields to remove/override
my %remove;
my %override;

my $substvars = Dpkg::Substvars->new();
my $tar_ignore_default_pattern_done;
my $diff_ignore_regex = get_default_diff_ignore_regex();

my @options;
my @cmdline_options;
while (@ARGV && $ARGV[0] =~ m/^-/) {
    my $arg = shift @ARGV;

    if ($arg eq '-b' or $arg eq '--build') {
        setopmode('build');
    } elsif ($arg eq '-x' or $arg eq '--extract') {
        setopmode('extract');
    } elsif ($arg eq '--before-build') {
        setopmode('before-build');
    } elsif ($arg eq '--after-build') {
        setopmode('after-build');
    } elsif ($arg eq '--commit') {
        setopmode('commit');
    } elsif ($arg eq '--print-format') {
        setopmode('print-format');
	report_options(info_fh => \*STDERR); # Avoid clutter on STDOUT
    } else {
        push @options, $arg;
    }
}

my $dir;
if (defined($options{opmode}) &&
    $options{opmode} =~ /^(build|print-format|(before|after)-build|commit)$/) {
    if (not scalar(@ARGV)) {
	usageerr(g_('--%s needs a directory'), $options{opmode})
	    unless $1 eq 'commit';
	$dir = '.';
    } else {
	$dir = File::Spec->catdir(shift(@ARGV));
    }
    stat($dir) or syserr(g_('cannot stat directory %s'), $dir);
    if (not -d $dir) {
	error(g_('directory argument %s is not a directory'), $dir);
    }
    if ($dir eq '.') {
	# . is never correct, adjust automatically
	$dir = basename(getcwd());
	chdir '..' or syserr(g_("unable to chdir to '%s'"), '..');
    }
    # --format options are not allowed, they would take precedence
    # over real command line options, debian/source/format should be used
    # instead
    # --unapply-patches is only allowed in local-options as it's a matter
    # of personal taste and the default should be to keep patches applied
    my $forbidden_opts_re = {
	'options' => qr/^--(?:format=|unapply-patches$|abort-on-upstream-changes$)/,
	'local-options' => qr/^--format=/,
    };
    foreach my $filename ('local-options', 'options') {
	my $conf = Dpkg::Conf->new();
	my $optfile = File::Spec->catfile($dir, 'debian', 'source', $filename);
	next unless -f $optfile;
	$conf->load($optfile);
	$conf->filter(remove => sub { $_[0] =~ $forbidden_opts_re->{$filename} });
	if (@$conf) {
	    info(g_('using options from %s: %s'), $optfile, join(' ', @$conf))
		unless $options{opmode} eq 'print-format';
	    unshift @options, @$conf;
	}
    }
}

while (@options) {
    $_ = shift(@options);
    if (m/^--format=(.*)$/) {
	$build_format //= $1;
    } elsif (m/^-(?:Z|-compression=)(.*)$/) {
	my $compression = $1;
	$options{compression} = $compression;
	usageerr(g_('%s is not a supported compression'), $compression)
	    unless compression_is_supported($compression);
	compression_set_default($compression);
    } elsif (m/^-(?:z|-compression-level=)(.*)$/) {
	my $comp_level = $1;
	$options{comp_level} = $comp_level;
	usageerr(g_('%s is not a compression level'), $comp_level)
	    unless compression_is_valid_level($comp_level);
	compression_set_default_level($comp_level);
    } elsif (m/^-c(.*)$/) {
        $controlfile = $1;
    } elsif (m/^-l(.*)$/) {
        $changelogfile = $1;
    } elsif (m/^-F([0-9a-z]+)$/) {
        $changelogformat = $1;
    } elsif (m/^-D([^\=:]+)[=:](.*)$/s) {
        $override{$1} = $2;
    } elsif (m/^-U([^\=:]+)$/) {
        $remove{$1} = 1;
    } elsif (m/^-(?:i|-diff-ignore(?:$|=))(.*)$/) {
        $options{diff_ignore_regex} = $1 ? $1 : $diff_ignore_regex;
    } elsif (m/^--extend-diff-ignore=(.+)$/) {
	$diff_ignore_regex .= "|$1";
	if ($options{diff_ignore_regex}) {
	    $options{diff_ignore_regex} .= "|$1";
	}
	set_default_diff_ignore_regex($diff_ignore_regex);
    } elsif (m/^-(?:I|-tar-ignore=)(.+)$/) {
        push @{$options{tar_ignore}}, $1;
    } elsif (m/^-(?:I|-tar-ignore)$/) {
        unless ($tar_ignore_default_pattern_done) {
            push @{$options{tar_ignore}}, get_default_tar_ignore_pattern();
            # Prevent adding multiple times
            $tar_ignore_default_pattern_done = 1;
        }
    } elsif (m/^--no-copy$/) {
        $options{copy_orig_tarballs} = 0;
    } elsif (m/^--no-check$/) {
        $options{no_check} = 1;
    } elsif (m/^--no-overwrite-dir$/) {
        $options{no_overwrite_dir} = 1;
    } elsif (m/^--require-valid-signature$/) {
        $options{require_valid_signature} = 1;
    } elsif (m/^--require-strong-checksums$/) {
        $options{require_strong_checksums} = 1;
    } elsif (m/^-V(\w[-:0-9A-Za-z]*)[=:](.*)$/s) {
        $substvars->set($1, $2);
    } elsif (m/^-T(.*)$/) {
	$substvars->load($1) if -e $1;
    } elsif (m/^-(?:\?|-help)$/) {
        usage();
        exit(0);
    } elsif (m/^--version$/) {
        version();
        exit(0);
    } elsif (m/^-[EW]$/) {
        # Deprecated option
        warning(g_('-E and -W are deprecated, they are without effect'));
    } elsif (m/^-q$/) {
        report_options(quiet_warnings => 1);
        $options{quiet} = 1;
    } elsif (m/^--$/) {
        last;
    } else {
        push @cmdline_options, $_;
    }
}

unless (defined($options{opmode})) {
    usageerr(g_('need an action option'));
}

if ($options{opmode} =~ /^(build|print-format|(before|after)-build|commit)$/) {

    $options{ARGV} = \@ARGV;

    $changelogfile ||= "$dir/debian/changelog";
    $controlfile ||= "$dir/debian/control";

    my %ch_options = (file => $changelogfile);
    $ch_options{changelogformat} = $changelogformat if $changelogformat;
    my $changelog = changelog_parse(%ch_options);
    my $control = Dpkg::Control::Info->new($controlfile);

    # <https://reproducible-builds.org/specs/source-date-epoch/>
    $ENV{SOURCE_DATE_EPOCH} ||= $changelog->{timestamp} || time;

    # Select the format to use
    if (not defined $build_format) {
        my $format_file = "$dir/debian/source/format";
        if (-e $format_file) {
            my $format = Dpkg::Source::Format->new(filename => $format_file);
            $build_format = $format->get();
        } else {
            warning(g_('no source format specified in %s, ' .
                       'see dpkg-source(1)'), 'debian/source/format')
                if $options{opmode} eq 'build';
            $build_format = '1.0';
        }
    }

    my $srcpkg = Dpkg::Source::Package->new(format => $build_format,
                                            options => \%options);
    my $fields = $srcpkg->{fields};

    $srcpkg->parse_cmdline_options(@cmdline_options);

    my @sourcearch;
    my %archadded;
    my @binarypackages;

    # Scan control info of source package
    my $src_fields = $control->get_source();
    error(g_("%s doesn't contain any information about the source package"),
          $controlfile) unless defined $src_fields;
    my $src_sect = $src_fields->{'Section'} || 'unknown';
    my $src_prio = $src_fields->{'Priority'} || 'unknown';
    foreach (keys %{$src_fields}) {
	my $v = $src_fields->{$_};
	if (m/^Source$/i) {
	    set_source_package($v);
	    $fields->{$_} = $v;
	} elsif (m/^Uploaders$/i) {
	    ($fields->{$_} = $v) =~ s/\s*[\r\n]\s*/ /g; # Merge in a single-line
	} elsif (m/^Build-(?:Depends|Conflicts)(?:-Arch|-Indep)?$/i) {
	    my $dep;
	    my $type = field_get_dep_type($_);
	    $dep = deps_parse($v, build_dep => 1, union => $type eq 'union');
	    error(g_('cannot parse %s field'), $_) unless defined $dep;
	    my $facts = Dpkg::Deps::KnownFacts->new();
	    $dep->simplify_deps($facts);
	    $dep->sort() if $type eq 'union';
	    $fields->{$_} = $dep->output();
	} else {
            field_transfer_single($src_fields, $fields);
	}
    }

    # Scan control info of binary packages
    my @pkglist;
    foreach my $pkg ($control->get_packages()) {
	my $p = $pkg->{'Package'};
	my $sect = $pkg->{'Section'} || $src_sect;
	my $prio = $pkg->{'Priority'} || $src_prio;
	my $type = $pkg->{'Package-Type'} ||
	        $pkg->get_custom_field('Package-Type') || 'deb';
        my $arch = $pkg->{'Architecture'};
        my $profile = $pkg->{'Build-Profiles'};

        my $pkg_summary = sprintf('%s %s %s %s', $p, $type, $sect, $prio);

        $pkg_summary .= ' arch=' . join ',', split ' ', $arch;

        if (defined $profile) {
            # Instead of splitting twice and then joining twice, we just do
            # simple string replacements:

            # Remove the enclosing <>
            $profile =~ s/^\s*<(.*)>\s*$/$1/;
            # Join lists with a plus (OR)
            $profile =~ s/>\s+</+/g;
            # Join their elements with a comma (AND)
            $profile =~ s/\s+/,/g;
            $pkg_summary .= " profile=$profile";
        }
        if (defined $pkg->{'Protected'} and $pkg->{'Protected'} eq 'yes') {
            $pkg_summary .= ' protected=yes';
        }
        if (defined $pkg->{'Essential'} and $pkg->{'Essential'} eq 'yes') {
            $pkg_summary .= ' essential=yes';
        }

        push @pkglist, $pkg_summary;
	push @binarypackages, $p;
	foreach (keys %{$pkg}) {
	    my $v = $pkg->{$_};
            if (m/^Architecture$/) {
                # Gather all binary architectures in one set. 'any' and 'all'
                # are special-cased as they need to be the only ones in the
                # current stanza if present.
                if (debarch_eq($v, 'any') || debarch_eq($v, 'all')) {
                    push(@sourcearch, $v) unless $archadded{$v}++;
                } else {
                    for my $a (split(/\s+/, $v)) {
                        error(g_("'%s' is not a legal architecture string " .
                                 "in package '%s'"), $a, $p)
                            if debarch_is_illegal($a);
                        error(g_('architecture %s only allowed on its ' .
                                 "own (list for package %s is '%s')"),
                              $a, $p, $a)
                            if $a eq 'any' or $a eq 'all';
                        push(@sourcearch, $a) unless $archadded{$a}++;
                    }
                }
            } elsif (m/^(?:Homepage|Description)$/) {
                # Do not overwrite the same field from the source entry
            } else {
                field_transfer_single($pkg, $fields);
            }
	}
    }
    unless (scalar(@pkglist)) {
	error(g_("%s doesn't list any binary package"), $controlfile);
    }
    if (any { $_ eq 'any' } @sourcearch) {
        # If we encounter one 'any' then the other arches become insignificant
        # except for 'all' that must also be kept
        if (any { $_ eq 'all' } @sourcearch) {
            @sourcearch = qw(any all);
        } else {
            @sourcearch = qw(any);
        }
    } else {
        # Minimize arch list, by removing arches already covered by wildcards
        my @arch_wildcards = grep { debarch_is_wildcard($_) } @sourcearch;
        my @mini_sourcearch = @arch_wildcards;
        foreach my $arch (@sourcearch) {
            if (none { debarch_is($arch, $_) } @arch_wildcards) {
                push @mini_sourcearch, $arch;
            }
        }
        @sourcearch = @mini_sourcearch;
    }
    $fields->{'Architecture'} = join(' ', @sourcearch);
    $fields->{'Package-List'} = "\n" . join("\n", sort @pkglist);

    # Check if we have a testsuite, and handle manual and automatic values.
    set_testsuite_fields($fields, @binarypackages);

    # Scan fields of dpkg-parsechangelog
    foreach (keys %{$changelog}) {
        my $v = $changelog->{$_};

	if (m/^Source$/) {
	    set_source_package($v);
	    $fields->{$_} = $v;
	} elsif (m/^Version$/) {
	    my ($ok, $error) = version_check($v);
            error($error) unless $ok;
	    $fields->{$_} = $v;
	} elsif (m/^Binary-Only$/) {
	    error(g_('building source for a binary-only release'))
	        if $v eq 'yes' and $options{opmode} eq 'build';
	} elsif (m/^Maintainer$/i) {
            # Do not replace the field coming from the source entry
	} else {
            field_transfer_single($changelog, $fields);
	}
    }

    $fields->{'Binary'} = join(', ', @binarypackages);
    # Avoid overly long line by splitting over multiple lines
    if (length($fields->{'Binary'}) > 980) {
	$fields->{'Binary'} =~ s/(.{0,980}), ?/$1,\n/g;
    }

    if ($options{opmode} eq 'print-format') {
	print $fields->{'Format'} . "\n";
	exit(0);
    } elsif ($options{opmode} eq 'before-build') {
	$srcpkg->before_build($dir);
	exit(0);
    } elsif ($options{opmode} eq 'after-build') {
	$srcpkg->after_build($dir);
	exit(0);
    } elsif ($options{opmode} eq 'commit') {
	$srcpkg->commit($dir);
	exit(0);
    }

    # Verify pre-requisites are met
    my ($res, $msg) = $srcpkg->can_build($dir);
    error(g_("can't build with source format '%s': %s"), $build_format, $msg) unless $res;

    # Only -b left
    info(g_("using source format '%s'"), $fields->{'Format'});
    run_vendor_hook('before-source-build', $srcpkg);
    # Build the files (.tar.gz, .diff.gz, etc)
    $srcpkg->build($dir);

    # Write the .dsc
    my $dscname = $srcpkg->get_basename(1) . '.dsc';
    info(g_('building %s in %s'), get_source_package(), $dscname);
    $srcpkg->write_dsc(filename => $dscname,
		       remove => \%remove,
		       override => \%override,
		       substvars => $substvars);
    exit(0);

} elsif ($options{opmode} eq 'extract') {

    # Check command line
    unless (scalar(@ARGV)) {
        usageerr(g_('--%s needs at least one argument, the .dsc'),
                 $options{opmode});
    }
    if (scalar(@ARGV) > 2) {
        usageerr(g_('--%s takes no more than two arguments'), $options{opmode});
    }
    my $dsc = shift(@ARGV);
    if (-d $dsc) {
        usageerr(g_('--%s needs the .dsc file as first argument, not a directory'),
                 $options{opmode});
    }

    # Create the object that does everything
    my $srcpkg = Dpkg::Source::Package->new(filename => $dsc,
					    options => \%options);

    # Parse command line options
    $srcpkg->parse_cmdline_options(@cmdline_options);

    # Decide where to unpack
    my $newdirectory = $srcpkg->get_basename();
    $newdirectory =~ s/_/-/g;
    if (@ARGV) {
	$newdirectory = File::Spec->catdir(shift(@ARGV));
	if (-e $newdirectory) {
	    error(g_('unpack target exists: %s'), $newdirectory);
	}
    }

    # Various checks before unpacking
    unless ($options{no_check}) {
        if ($srcpkg->is_signed()) {
            $srcpkg->check_signature();
        } else {
            if ($options{require_valid_signature}) {
                error(g_("%s doesn't contain a valid OpenPGP signature"), $dsc);
            } else {
                warning(g_('extracting unsigned source package (%s)'), $dsc);
            }
        }
        $srcpkg->check_checksums();
    }

    # Unpack the source package (delegated to Dpkg::Source::Package::*)
    info(g_('extracting %s in %s'), $srcpkg->{fields}{'Source'}, $newdirectory);
    $srcpkg->extract($newdirectory);

    exit(0);
}

sub set_testsuite_fields
{
    my ($fields, @binarypackages) = @_;

    my $testsuite_field = $fields->{'Testsuite'} // '';
    my %testsuite = map { $_ => 1 } split /\s*,\s*/, $testsuite_field;
    if (-e "$dir/debian/tests/control") {
        error(g_('test control %s is not a regular file'),
              'debian/tests/control') unless -f _;
        $testsuite{autopkgtest} = 1;

        my $tests = Dpkg::Control::Tests->new();
        $tests->load("$dir/debian/tests/control");

        set_testsuite_triggers_field($tests, $fields, @binarypackages);
    } elsif ($testsuite{autopkgtest}) {
        warning(g_('%s field contains value %s, but no tests control file %s'),
                'Testsuite', 'autopkgtest', 'debian/tests/control');
        delete $testsuite{autopkgtest};
    }
    $fields->{'Testsuite'} = join ', ', sort keys %testsuite;
}

sub set_testsuite_triggers_field
{
    my ($tests, $fields, @binarypackages) = @_;
    my %testdeps;

    # Never overwrite a manually defined field.
    return if $fields->{'Testsuite-Triggers'};

    foreach my $test ($tests->get()) {
        if (not exists $test->{Tests} and not exists $test->{'Test-Command'}) {
            error(g_('test control %s is missing %s or %s field'),
                  'debian/tests/control', 'Tests', 'Test-Command');
        }

        next unless $test->{Depends};

        my $deps = deps_parse($test->{Depends}, use_arch => 0, tests_dep => 1);
        deps_iterate($deps, sub { $testdeps{$_[0]->{package}} = 1 });
    }

    # Remove our own binaries and its meta-depends variant.
    foreach my $pkg (@binarypackages, qw(@)) {
        delete $testdeps{$pkg};
    }
    $fields->{'Testsuite-Triggers'} = join ', ', sort keys %testdeps;
}

sub setopmode {
    my $opmode = shift;

    if (defined($options{opmode})) {
        usageerr(g_('two commands specified: --%s and --%s'),
                 $options{opmode}, $opmode);
    }
    $options{opmode} = $opmode;
}

sub print_option {
    my $opt = shift;
    my $help;

    if (length $opt->{name} > 25) {
        $help .= sprintf "  %-25s\n%s%s.\n", $opt->{name}, ' ' x 27, $opt->{help};
    } else {
        $help .= sprintf "  %-25s%s.\n", $opt->{name}, $opt->{help};
    }
}

sub get_format_help {
    $build_format //= '1.0';

    my $srcpkg = Dpkg::Source::Package->new(format => $build_format);

    my @cmdline = $srcpkg->describe_cmdline_options();
    return '' unless @cmdline;

    my $help_build = my $help_extract = '';
    my $help;

    foreach my $opt (@cmdline) {
        $help_build .= print_option($opt) if $opt->{when} eq 'build';
        $help_extract .= print_option($opt) if $opt->{when} eq 'extract';
    }

    if ($help_build) {
        $help .= "\n";
        $help .= "Build format $build_format options:\n";
        $help .= $help_build || C_('source options', '<none>');
    }
    if ($help_extract) {
        $help .= "\n";
        $help .= "Extract format $build_format options:\n";
        $help .= $help_extract || C_('source options', '<none>');
    }

    return $help;
}

sub version {
    printf g_("Debian %s version %s.\n"), $Dpkg::PROGNAME, $Dpkg::PROGVERSION;

    print g_('
This is free software; see the GNU General Public License version 2 or
later for copying conditions. There is NO warranty.
');
}

sub usage {
    printf g_(
'Usage: %s [<option>...] <command>')
    . "\n\n" . g_(
'Commands:
  -x, --extract <filename>.dsc [<output-dir>]
                           extract source package.
  -b, --build <dir>        build source package.
      --print-format <dir> print the format to be used for the source package.
      --before-build <dir> run the corresponding source package format hook.
      --after-build <dir>  run the corresponding source package format hook.
      --commit [<dir> [<patch-name>]]
                           store upstream changes in a new patch.')
    . "\n\n" . g_(
"Build options:
  -c<control-file>         get control info from this file.
  -l<changelog-file>       get per-version info from this file.
  -F<changelog-format>     force changelog format.
  --format=<source-format> set the format to be used for the source package.
  -V<name>=<value>         set a substitution variable.
  -T<substvars-file>       read variables here.
  -D<field>=<value>        override or add a .dsc field and value.
  -U<field>                remove a field.
  -i, --diff-ignore[=<regex>]
                           filter out files to ignore diffs of
                             (defaults to: '%s').
  -I, --tar-ignore[=<pattern>]
                           filter out files when building tarballs
                             (defaults to: %s).
  -Z, --compression=<compression>
                           select compression to use (defaults to '%s',
                             supported are: %s).
  -z, --compression-level=<level>
                           compression level to use (defaults to '%d',
                             supported are: '1'-'9', 'best', 'fast')")
    . "\n\n" . g_(
"Extract options:
  --no-copy                don't copy .orig tarballs
  --no-check               don't check signature and checksums before unpacking
  --no-overwrite-dir       do not overwrite directory on extraction
  --require-valid-signature abort if the package doesn't have a valid signature
  --require-strong-checksums
                           abort if the package contains no strong checksums
  --ignore-bad-version     allow bad source package versions.")
    . "\n" .
    get_format_help()
    . "\n" . g_(
'General options:
  -q                       quiet mode.
  -?, --help               show this help message.
      --version            show the version.')
    . "\n\n" . g_(
'Source format specific build and extract options are available;
use --format with --help to see them.') . "\n",
    $Dpkg::PROGNAME,
    get_default_diff_ignore_regex(),
    join(' ', map { "-I$_" } get_default_tar_ignore_pattern()),
    compression_get_default(),
    join(' ', compression_get_list()),
    compression_get_default_level();
}
