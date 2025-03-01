#!/usr/bin/perl
#
# dpkg-buildflags
#
# Copyright © 2010-2011 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2012-2013 Guillem Jover <guillem@debian.org>
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

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::ErrorHandling qw(:DEFAULT report REPORT_STATUS);
use Dpkg::Build::Env;
use Dpkg::BuildFlags;
use Dpkg::Vendor qw(get_current_vendor);

textdomain('dpkg-dev');

sub version {
    printf g_("Debian %s version %s.\n"), $Dpkg::PROGNAME, $Dpkg::PROGVERSION;

    printf g_('
This is free software; see the GNU General Public License version 2 or
later for copying conditions. There is NO warranty.
');
}

sub usage {
    printf g_(
'Usage: %s [<command>]')
    . "\n\n" . g_(
'Commands:
  --get <flag>       output the requested flag to stdout.
  --origin <flag>    output the origin of the flag to stdout:
                     value is one of vendor, system, user, env.
  --status           output a synopsis with all parameters affecting the
                     program behaviour, the resulting flags and their origin.
  --query            like --status, but in deb822 format.
  --query-features <area>
                     output the status of features for the given area.
  --list             output a list of the flags supported by the current vendor.
  --export=(sh|make|cmdline|configure)
                     output something convenient to import the compilation
                     flags in a shell script, in make, or in a command line.
  --dump             output all compilation flags with their values.
  --help             show this help message.
  --version          show the version.
'), $Dpkg::PROGNAME;
}

my ($param, $action);
my $load_config = 1;

while (@ARGV) {
    $_ = shift(@ARGV);
    if (m/^--(get|origin|query-features)$/) {
        usageerr(g_('two commands specified: --%s and --%s'), $1, $action)
            if defined($action);
        $action = $1;
        $param = shift(@ARGV);
	usageerr(g_('%s needs a parameter'), $_) unless defined $param;
    } elsif (m/^--export(?:=(sh|make|cmdline|configure))?$/) {
        usageerr(g_('two commands specified: --%s and --%s'), 'export', $action)
            if defined($action);
        my $type = $1 || 'sh';
        # Map legacy aliases.
        $type = 'cmdline' if $type eq 'configure';
        $action = "export-$type";
    } elsif (m/^--(list|status|dump|query)$/) {
        usageerr(g_('two commands specified: --%s and --%s'), $1, $action)
            if defined($action);
        $action = $1;
        $load_config = 0 if $action eq 'list';
    } elsif (m/^-(?:\?|-help)$/) {
        usage();
        exit 0;
    } elsif (m/^--version$/) {
        version();
        exit 0;
    } else {
        usageerr(g_("unknown option '%s'"), $_);
    }
}

$action //= 'dump';

my $build_flags = Dpkg::BuildFlags->new();

$build_flags->load_config() if $load_config;

if ($action eq 'list') {
    foreach my $flag ($build_flags->list()) {
	print "$flag\n";
    }
} elsif ($action eq 'get') {
    exit 1 unless $build_flags->has($param);

    print $build_flags->get($param) . "\n";
} elsif ($action eq 'origin') {
    exit 1 unless $build_flags->has($param);

    print $build_flags->get_origin($param) . "\n";
} elsif ($action eq 'query-features') {
    exit 1 unless $build_flags->has_features($param);

    my %features = $build_flags->get_features($param);
    my $para_shown = 0;
    foreach my $feature (sort keys %features) {
        print $para_shown++ ? "\n" : '';
        printf "Feature: %s\n", $feature;
        printf "Enabled: %s\n", $features{$feature} ? 'yes' : 'no';
    }
} elsif ($action =~ m/^export-(.*)$/) {
    my $export_type = $1;
    foreach my $flag ($build_flags->list()) {
	next unless $flag =~ /^[A-Z]/; # Skip flags starting with lowercase
	my $value = $build_flags->get($flag);
	if ($export_type eq 'sh') {
	    $value =~ s/"/\"/g;
	    print "export $flag=\"$value\"\n";
	} elsif ($export_type eq 'make') {
	    $value =~ s/\$/\$\$/g;
	    print "export $flag := $value\n";
	} elsif ($export_type eq 'cmdline') {
	    print "$flag=\"$value\" ";
	}
    }
} elsif ($action eq 'dump') {
    foreach my $flag ($build_flags->list()) {
	my $value = $build_flags->get($flag);
	print "$flag=$value\n";
    }
} elsif ($action eq 'query') {
    # First print all environment variables that might have changed the
    # results (only existing ones, might make sense to add an option to
    # also show which ones could have set to modify it).
    printf "Vendor: %s\n", Dpkg::Vendor::get_current_vendor() || 'undefined';
    print "Environment:\n";
    for my $envvar (Dpkg::Build::Env::list_accessed()) {
        print " $envvar=$ENV{$envvar}\n" if exists $ENV{$envvar};
    }

    # Then the resulting features:
    foreach my $area (sort $build_flags->get_feature_areas()) {
        print "\n";
        print "Area: $area\n";
        print "Features:\n";
        my %features = $build_flags->get_features($area);
        foreach my $feature (sort keys %features) {
            printf " %s=%s\n", $feature, $features{$feature} ? 'yes' : 'no';
        }
    }

    # Then the resulting values (with their origin):
    foreach my $flag ($build_flags->list()) {
        print "\n";
        print "Flag: $flag\n";
        printf "Value: %s\n", $build_flags->get($flag);
        my $origin = $build_flags->get_origin($flag);
        if ($build_flags->is_maintainer_modified($flag)) {
            $origin .= '+maintainer';
        }
        print "Origin: $origin\n";
    }
} elsif ($action eq 'status') {
    # Prefix everything with "dpkg-buildflags: status: " to allow easy
    # extraction from a build log. Thus we use report with a non-translated
    # type string.

    # First print all environment variables that might have changed the
    # results (only existing ones, might make sense to add an option to
    # also show which ones could have set to modify it).
    my @envvars = Dpkg::Build::Env::list_accessed();
    for my $envvar (@envvars) {
	if (exists $ENV{$envvar}) {
	    printf report(REPORT_STATUS, 'environment variable %s=%s',
	           $envvar, $ENV{$envvar});
	}
    }
    my $vendor = Dpkg::Vendor::get_current_vendor() || 'undefined';
    print report(REPORT_STATUS, "vendor is $vendor");
    # Then the resulting features:
    foreach my $area (sort $build_flags->get_feature_areas()) {
	my $fs;
	my %features = $build_flags->get_features($area);
	foreach my $feature (sort keys %features) {
	    $fs .= sprintf(' %s=%s', $feature, $features{$feature} ? 'yes' : 'no');
	}
	print report(REPORT_STATUS, "$area features:$fs");
    }
    # Then the resulting values (with their origin):
    foreach my $flag ($build_flags->list()) {
	my $value = $build_flags->get($flag);
	my $origin = $build_flags->get_origin($flag);
	my $maintainer = $build_flags->is_maintainer_modified($flag) ? '+maintainer' : '';
	print report(REPORT_STATUS, "$flag [$origin$maintainer]: $value");
    }
}
