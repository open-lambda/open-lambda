#!/usr/bin/perl
#
# dpkg-scanpackages
#
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

use warnings;
use strict;

use Getopt::Long qw(:config posix_default bundling_values no_ignorecase);
use List::Util qw(none);
use File::Find;

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Control;
use Dpkg::Version;
use Dpkg::Checksums;
use Dpkg::Compression::FileHandle;

textdomain('dpkg-dev');

# Do not pollute STDOUT with info messages
report_options(info_fh => \*STDERR);

my (@samemaint, @changedmaint);
my @multi_instances;
my @spuriousover;
my %packages;
my %overridden;
my @checksums;

my %options = (help            => sub { usage(); exit 0; },
	       version         => sub { version(); exit 0; },
	       type            => undef,
	       arch            => undef,
	       hash            => undef,
	       multiversion    => 0,
	       'extra-override'=> undef,
               medium          => undef,
	      );

my @options_spec = (
    'help|?',
    'version',
    'type|t=s',
    'arch|a=s',
    'hash|h=s',
    'multiversion|m!',
    'extra-override|e=s',
    'medium|M=s',
);

sub version {
    printf g_("Debian %s version %s.\n"), $Dpkg::PROGNAME, $Dpkg::PROGVERSION;
}

sub usage {
    printf g_(
"Usage: %s [<option>...] <binary-path> [<override-file> [<path-prefix>]] > Packages

Options:
  -t, --type <type>        scan for <type> packages (default is 'deb').
  -a, --arch <arch>        architecture to scan for.
  -h, --hash <hash-list>   only generate hashes for the specified list.
  -m, --multiversion       allow multiple versions of a single package.
  -e, --extra-override <file>
                           use extra override file.
  -M, --medium <medium>    add X-Medium field for dselect multicd access method
  -?, --help               show this help message.
      --version            show the version.
"), $Dpkg::PROGNAME;
}

sub load_override
{
    my $override = shift;
    my $comp_file = Dpkg::Compression::FileHandle->new(filename => $override);

    while (<$comp_file>) {
	s/\#.*//;
	s/\s+$//;
	next unless $_;

	my ($p, $priority, $section, $maintainer) = split(/\s+/, $_, 4);

	if (not defined($packages{$p})) {
	    push(@spuriousover, $p);
	    next;
	}

	for my $package (@{$packages{$p}}) {
	    if ($maintainer) {
		if ($maintainer =~ m/(.+?)\s*=\>\s*(.+)/) {
		    my $oldmaint = $1;
		    my $newmaint = $2;
		    my $debmaint = $$package{Maintainer};
		    if (none { $debmaint eq $_ } split m{\s*//\s*}, $oldmaint) {
			push(@changedmaint,
			     sprintf(g_('  %s (package says %s, not %s)'),
			             $p, $$package{Maintainer}, $oldmaint));
		    } else {
			$$package{Maintainer} = $newmaint;
		    }
		} elsif ($$package{Maintainer} eq $maintainer) {
		    push(@samemaint, "  $p ($maintainer)");
		} else {
		    warning(g_('unconditional maintainer override for %s'), $p);
		    $$package{Maintainer} = $maintainer;
		}
	    }
	    $$package{Priority} = $priority;
	    $$package{Section} = $section;
	}
	$overridden{$p} = 1;
    }

    close($comp_file);
}

sub load_override_extra
{
    my $extra_override = shift;
    my $comp_file = Dpkg::Compression::FileHandle->new(filename => $extra_override);

    while (<$comp_file>) {
	s/\#.*//;
	s/\s+$//;
	next unless $_;

	my ($p, $field, $value) = split(/\s+/, $_, 3);

	next unless defined($packages{$p});

	for my $package (@{$packages{$p}}) {
	    $$package{$field} = $value;
	}
    }

    close($comp_file);
}

sub process_deb {
    my ($pathprefix, $fn) = @_;

    my $fields = Dpkg::Control->new(type => CTRL_INDEX_PKG);

    open my $output_fh, '-|', 'dpkg-deb', '-I', $fn, 'control'
        or syserr(g_('cannot fork for %s'), 'dpkg-deb');
    $fields->parse($output_fh, $fn)
        or error(g_("couldn't parse control information from %s"), $fn);
    close $output_fh;
    if ($?) {
        warning(g_("'dpkg-deb -I %s control' exited with %d, skipping package"),
                $fn, $?);
        return;
    }

    my $p = $fields->{'Package'};
    error(g_('no Package field in control file of %s'), $fn)
        if not defined $p;

    if (defined($packages{$p}) and not $options{multiversion}) {
        my $pkg = ${$packages{$p}}[0];

        @multi_instances = ($pkg->{Filename}) if @multi_instances == 0;
        push @multi_instances, "$pathprefix$fn";

        if (version_compare_relation($fields->{'Version'}, REL_GT,
                                     $pkg->{'Version'}))
        {
            warning(g_('package %s (filename %s) is repeat but newer ' .
                       'version; used that one and ignored data from %s!'),
                    $p, $fn, $pkg->{Filename});
            $packages{$p} = [];
        } else {
            warning(g_('package %s (filename %s) is repeat; ' .
                       'ignored that one and using data from %s!'),
                    $p, $fn, $pkg->{Filename});
            return;
        }
    }

    warning(g_('package %s (filename %s) has Filename field!'), $p, $fn)
        if defined($fields->{'Filename'});
    $fields->{'Filename'} = "$pathprefix$fn";

    my $sums = Dpkg::Checksums->new();
    $sums->add_from_file($fn, checksums => \@checksums);
    foreach my $alg (@checksums) {
        if ($alg eq 'md5') {
            $fields->{'MD5sum'} = $sums->get_checksum($fn, $alg);
        } else {
            $fields->{$alg} = $sums->get_checksum($fn, $alg);
        }
    }
    $fields->{'Size'} = $sums->get_size($fn);
    $fields->{'X-Medium'} = $options{medium} if defined $options{medium};

    push @{$packages{$p}}, $fields;
}

{
    local $SIG{__WARN__} = sub { usageerr($_[0]) };
    GetOptions(\%options, @options_spec);
}

if (not (@ARGV >= 1 and @ARGV <= 3)) {
    usageerr(g_('one to three arguments expected'));
}

my $type = $options{type} // 'deb';
my $arch = $options{arch};
my %hash = map { $_ => 1 } split /,/, $options{hash} // '';

foreach my $alg (keys %hash) {
    if (not checksums_is_supported($alg)) {
        usageerr(g_('unsupported checksum \'%s\''), $alg);
    }
}
@checksums = %hash ? keys %hash : checksums_get_list();

my ($binarypath, $override, $pathprefix) = @ARGV;

if (not -e $binarypath) {
    error(g_('binary path %s not found'), $binarypath);
}
if (defined $override and not -e $override) {
    error(g_('override file %s not found'), $override);
}

$pathprefix //= '';

my $find_filter;
if ($options{arch}) {
    $find_filter = qr/_(?:all|${arch})\.$type$/;
} else {
    $find_filter = qr/\.$type$/;
}
my @archives;
my $scan_archives = sub {
    push @archives, $File::Find::name if m/$find_filter/;
};

find({ follow => 1, follow_skip => 2, wanted => $scan_archives}, $binarypath);
foreach my $fn (@archives) {
    process_deb($pathprefix, $fn);
}

load_override($override) if defined $override;
load_override_extra($options{'extra-override'}) if defined $options{'extra-override'};

my @missingover=();

my $records_written = 0;
for my $p (sort keys %packages) {
    if (defined($override) and not defined($overridden{$p})) {
        push @missingover, $p;
    }
    for my $package (sort { $a->{Version} cmp $b->{Version} } @{$packages{$p}}) {
         print("$package\n") or syserr(g_('failed when writing stdout'));
         $records_written++;
    }
}
close(STDOUT) or syserr(g_("couldn't close stdout"));

if (@multi_instances) {
    warning(g_('Packages with multiple instances but no --multiversion specified:'));
    warning($_) foreach (sort @multi_instances);
}
if (@changedmaint) {
    warning(g_('Packages in override file with incorrect old maintainer value:'));
    warning($_) foreach (@changedmaint);
}
if (@samemaint) {
    warning(g_('Packages specifying same maintainer as override file:'));
    warning($_) foreach (@samemaint);
}
if (@missingover) {
    warning(g_('Packages in archive but missing from override file:'));
    warning('  %s', join(' ', @missingover));
}
if (@spuriousover) {
    warning(g_('Packages in override file but not in archive:'));
    warning('  %s', join(' ', @spuriousover));
}

info(g_('Wrote %s entries to output Packages file.'), $records_written);
