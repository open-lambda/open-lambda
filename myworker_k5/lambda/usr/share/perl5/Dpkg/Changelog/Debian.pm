# Copyright © 1996 Ian Jackson
# Copyright © 2005 Frank Lichtenheld <frank@lichtenheld.de>
# Copyright © 2009 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2012-2017 Guillem Jover <guillem@debian.org>
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

=encoding utf8

=head1 NAME

Dpkg::Changelog::Debian - parse Debian changelogs

=head1 DESCRIPTION

This class represents a Debian changelog file as an array of changelog
entries (Dpkg::Changelog::Entry::Debian).
It implements the generic interface Dpkg::Changelog.
Only methods specific to this implementation are described below,
the rest are inherited.

Dpkg::Changelog::Debian parses Debian changelogs as described in
deb-changelog(5).

The parser tries to ignore most cruft like # or /* */ style comments,
RCS keywords, Vim modelines, Emacs local variables and stuff from
older changelogs with other formats at the end of the file.
NOTE: most of these are ignored silently currently, there is no
parser error issued for them. This should become configurable in the
future.

=cut

package Dpkg::Changelog::Debian;

use strict;
use warnings;

our $VERSION = '1.00';

use Dpkg::Gettext;
use Dpkg::File;
use Dpkg::Changelog qw(:util);
use Dpkg::Changelog::Entry::Debian qw(match_header match_trailer);

use parent qw(Dpkg::Changelog);

use constant {
    FIRST_HEADING => g_('first heading'),
    NEXT_OR_EOF => g_('next heading or end of file'),
    START_CHANGES => g_('start of change data'),
    CHANGES_OR_TRAILER => g_('more change data or trailer'),
};

my $ancient_delimiter_re = qr{
    ^
    (?: # Ancient GNU style changelog entry with expanded date
      (?:
        \w+\s+                          # Day of week (abbreviated)
        \w+\s+                          # Month name (abbreviated)
        \d{1,2}                         # Day of month
        \Q \E
        \d{1,2}:\d{1,2}:\d{1,2}\s+      # Time
        [\w\s]*                         # Timezone
        \d{4}                           # Year
      )
      \s+
      (?:.*)                            # Maintainer name
      \s+
      [<\(]
        (?:.*)                          # Maintainer email
      [\)>]
    | # Old GNU style changelog entry with expanded date
      (?:
        \w+\s+                          # Day of week (abbreviated)
        \w+\s+                          # Month name (abbreviated)
        \d{1,2},?\s*                    # Day of month
        \d{4}                           # Year
      )
      \s+
      (?:.*)                            # Maintainer name
      \s+
      [<\(]
        (?:.*)                          # Maintainer email
      [\)>]
    | # Ancient changelog header w/o key=value options
      (?:\w[-+0-9a-z.]*)                # Package name
      \Q \E
      \(
        (?:[^\(\) \t]+)                 # Package version
      \)
      \;?
    | # Ancient changelog header
      (?:[\w.+-]+)                      # Package name
      [- ]
      (?:\S+)                           # Package version
      \ Debian
      \ (?:\S+)                         # Package revision
    |
      Changes\ from\ version\ (?:.*)\ to\ (?:.*):
    |
      Changes\ for\ [\w.+-]+-[\w.+-]+:?\s*$
    |
      Old\ Changelog:\s*$
    |
      (?:\d+:)?
      \w[\w.+~-]*:?
      \s*$
    )
}xi;

=head1 METHODS

=over 4

=item $count = $c->parse($fh, $description)

Read the filehandle and parse a Debian changelog in it, to store the entries
as an array of Dpkg::Changelog::Entry::Debian objects.
Any previous entries in the object are reset before parsing new data.

Returns the number of changelog entries that have been parsed with success.

=cut

sub parse {
    my ($self, $fh, $file) = @_;
    $file = $self->{reportfile} if exists $self->{reportfile};

    $self->reset_parse_errors;

    $self->{data} = [];
    $self->set_unparsed_tail(undef);

    my $expect = FIRST_HEADING;
    my $entry = Dpkg::Changelog::Entry::Debian->new();
    my @blanklines = ();
    my $unknowncounter = 1; # to make version unique, e.g. for using as id
    local $_;

    while (<$fh>) {
	chomp;
	if (match_header($_)) {
	    unless ($expect eq FIRST_HEADING || $expect eq NEXT_OR_EOF) {
		$self->parse_error($file, $.,
		    sprintf(g_('found start of entry where expected %s'),
		    $expect), "$_");
	    }
	    unless ($entry->is_empty) {
		push @{$self->{data}}, $entry;
		$entry = Dpkg::Changelog::Entry::Debian->new();
		last if $self->abort_early();
	    }
	    $entry->set_part('header', $_);
	    foreach my $error ($entry->parse_header()) {
		$self->parse_error($file, $., $error, $_);
	    }
	    $expect= START_CHANGES;
	    @blanklines = ();
	} elsif (m/^(?:;;\s*)?Local variables:/io) {
            # Save any trailing Emacs variables at end of file.
            $self->set_unparsed_tail("$_\n" . (file_slurp($fh) // ''));
            last;
	} elsif (m/^vim:/io) {
            # Save any trailing Vim modelines at end of file.
            $self->set_unparsed_tail("$_\n" . (file_slurp($fh) // ''));
            last;
	} elsif (m/^\$\w+:.*\$/o) {
	    next; # skip stuff that look like a RCS keyword
	} elsif (m/^\# /o) {
	    next; # skip comments, even that's not supported
	} elsif (m{^/\*.*\*/}o) {
	    next; # more comments
	} elsif (m/$ancient_delimiter_re/) {
	    # save entries on old changelog format verbatim
	    # we assume the rest of the file will be in old format once we
	    # hit it for the first time
	    $self->set_unparsed_tail("$_\n" . file_slurp($fh));
	} elsif (m/^\S/) {
	    $self->parse_error($file, $., g_('badly formatted heading line'), "$_");
	} elsif (match_trailer($_)) {
	    unless ($expect eq CHANGES_OR_TRAILER) {
		$self->parse_error($file, $.,
		    sprintf(g_('found trailer where expected %s'), $expect), "$_");
	    }
	    $entry->set_part('trailer', $_);
	    $entry->extend_part('blank_after_changes', [ @blanklines ]);
	    @blanklines = ();
	    foreach my $error ($entry->parse_trailer()) {
		$self->parse_error($file, $., $error, $_);
	    }
	    $expect = NEXT_OR_EOF;
	} elsif (m/^ \-\-/) {
	    $self->parse_error($file, $., g_('badly formatted trailer line'), "$_");
	} elsif (m/^\s{2,}(?:\S)/) {
	    unless ($expect eq START_CHANGES or $expect eq CHANGES_OR_TRAILER) {
		$self->parse_error($file, $., sprintf(g_('found change data' .
		    ' where expected %s'), $expect), "$_");
		if ($expect eq NEXT_OR_EOF and not $entry->is_empty) {
		    # lets assume we have missed the actual header line
		    push @{$self->{data}}, $entry;
		    $entry = Dpkg::Changelog::Entry::Debian->new();
		    $entry->set_part('header', 'unknown (unknown' . ($unknowncounter++) . ') unknown; urgency=unknown');
		}
	    }
	    # Keep raw changes
	    $entry->extend_part('changes', [ @blanklines, $_ ]);
	    @blanklines = ();
	    $expect = CHANGES_OR_TRAILER;
	} elsif (!m/\S/) {
	    if ($expect eq START_CHANGES) {
		$entry->extend_part('blank_after_header', $_);
		next;
	    } elsif ($expect eq NEXT_OR_EOF) {
		$entry->extend_part('blank_after_trailer', $_);
		next;
	    } elsif ($expect ne CHANGES_OR_TRAILER) {
		$self->parse_error($file, $.,
		    sprintf(g_('found blank line where expected %s'), $expect));
	    }
	    push @blanklines, $_;
	} else {
	    $self->parse_error($file, $., g_('unrecognized line'), "$_");
	    unless ($expect eq START_CHANGES or $expect eq CHANGES_OR_TRAILER) {
		# lets assume change data if we expected it
		$entry->extend_part('changes', [ @blanklines, $_]);
		@blanklines = ();
		$expect = CHANGES_OR_TRAILER;
	    }
	}
    }

    unless ($expect eq NEXT_OR_EOF) {
        $self->parse_error($file, $.,
                           sprintf(g_('found end of file where expected %s'),
                                   $expect));
    }
    unless ($entry->is_empty) {
	push @{$self->{data}}, $entry;
    }

    return scalar @{$self->{data}};
}

1;
__END__

=back

=head1 CHANGES

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=head1 SEE ALSO

Dpkg::Changelog

=cut
