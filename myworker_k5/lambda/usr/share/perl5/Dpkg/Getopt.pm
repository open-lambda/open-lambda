# Copyright © 2014 Guillem Jover <guillem@debian.org>
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

package Dpkg::Getopt;

use strict;
use warnings;

our $VERSION = '0.02';
our @EXPORT = qw(
    normalize_options
);

use Exporter qw(import);

sub normalize_options
{
    my (%opts) = @_;
    my $norm = 1;
    my @args;

    @args = map {
        if ($norm and m/^(-[A-Za-z])(.+)$/) {
            ($1, $2)
        } elsif ($norm and m/^(--[A-Za-z-]+)=(.*)$/) {
            ($1, $2)
        } else {
            $norm = 0 if defined $opts{delim} and $_ eq $opts{delim};
            $_;
        }
    } @{$opts{args}};

    return @args;
}

1;
