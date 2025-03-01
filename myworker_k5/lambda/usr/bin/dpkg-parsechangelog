#!/usr/bin/perl
#
# dpkg-parsechangelog
#
# Copyright © 1996 Ian Jackson
# Copyright © 2001 Wichert Akkerman
# Copyright © 2006-2012 Guillem Jover <guillem@debian.org>
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
use Dpkg::Getopt;
use Dpkg::ErrorHandling;
use Dpkg::Changelog::Parse;

textdomain('dpkg-dev');

my %options;
my $fieldname;

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
  -l, --file <changelog-file>
                           get per-version info from this file.
  -F <changelog-format>    force changelog format.
  -S, --show-field <field> show the values for <field>.
  -?, --help               show this help message.
      --version            show the version.')
    . "\n\n" . g_(
"Parser options:
      --format <output-format>
                           set output format (defaults to 'dpkg').
      --reverse            include all changes in reverse order.
      --all                include all changes.
  -s, --since <version>    include all changes later than <version>.
  -v <version>             ditto.
  -u, --until <version>    include all changes earlier than <version>.
  -f, --from <version>     include all changes equal or later than <version>.
  -t, --to <version>       include all changes up to or equal than <version>.
  -c, --count <number>     include <number> entries from the top (or tail
                             if <number> is lower than 0).
  -n <number>              ditto.
  -o, --offset <number>    change starting point for --count, counted from
                             the top (or tail if <number> is lower than 0).
"), $Dpkg::PROGNAME;
}

@ARGV = normalize_options(args => \@ARGV, delim => '--');

while (@ARGV) {
    last unless $ARGV[0] =~ m/^-/;

    my $arg = shift;

    if ($arg eq '--') {
        last;
    } elsif ($arg eq '-L') {
        warning(g_('-L is obsolete; it is without effect'));
    } elsif ($arg eq '-F') {
        $options{changelogformat} = shift;
        usageerr(g_('bad changelog format name'))
            unless length $options{changelogformat} and
                          $options{changelogformat} =~ m/^([0-9a-z]+)$/;
    } elsif ($arg eq '--format') {
        $options{format} = shift;
    } elsif ($arg eq '--reverse') {
        $options{reverse} = 1;
    } elsif ($arg eq '-l' or $arg eq '--file') {
        $options{file} = shift;
        usageerr(g_('missing changelog filename'))
            unless length $options{file};
    } elsif ($arg eq '-S' or $arg eq '--show-field') {
        $fieldname = shift;
    } elsif ($arg eq '-c' or $arg eq '--count' or $arg eq '-n') {
        $options{count} = shift;
    } elsif ($arg eq '-f' or $arg eq '--from') {
        $options{from} = shift;
    } elsif ($arg eq '-o' or $arg eq '--offset') {
        $options{offset} = shift;
    } elsif ($arg eq '-s' or $arg eq '--since' or $arg eq '-v') {
        $options{since} = shift;
    } elsif ($arg eq '-t' or $arg eq '--to') {
        $options{to} = shift;
    } elsif ($arg eq '-u' or $arg eq '--until') {
        ## no critic (ControlStructures::ProhibitUntilBlocks)
        $options{until} = shift;
        ## use critic
    } elsif ($arg eq '--all') {
	$options{all} = undef;
    } elsif ($arg eq '-?' or $arg eq '--help') {
	usage(); exit(0);
    } elsif ($arg eq '--version') {
	version(); exit(0);
    } else {
	usageerr(g_("unknown option '%s'"), $arg);
    }
}
usageerr(g_('takes no non-option arguments')) if @ARGV;

my $count = 0;
my @fields = changelog_parse(%options);
foreach my $f (@fields) {
    print "\n" if $count++;
    if ($fieldname) {
        next if not exists $f->{$fieldname};

        my ($first_line, @lines) = split /\n/, $f->{$fieldname};

        my $v = '';
        $v .= $first_line if length $first_line;
        $v .= "\n";
        foreach (@lines) {
            s/\s+$//;
            if (length == 0 or /^\.+$/) {
                $v .= ".$_\n";
            } else {
                $v .= "$_\n";
            }
        }
        print $v;
    } else {
        print $f->output();
    }
}
