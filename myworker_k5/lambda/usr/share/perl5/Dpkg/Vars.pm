# Copyright © 2007-2009,2012-2013 Guillem Jover <guillem@debian.org>
# Copyright © 2007 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::Vars;

use strict;
use warnings;

our $VERSION = '0.03';
our @EXPORT = qw(
    get_source_package
    set_source_package
);

use Exporter qw(import);

use Dpkg::ErrorHandling;
use Dpkg::Gettext;
use Dpkg::Package;

my $sourcepackage;

sub get_source_package {
    return $sourcepackage;
}

sub set_source_package {
    my $v = shift;
    my $err = pkg_name_is_illegal($v);
    error(g_("source package name '%s' is illegal: %s"), $v, $err) if $err;

    if (not defined($sourcepackage)) {
        $sourcepackage = $v;
    } elsif ($v ne $sourcepackage) {
        error(g_('source package has two conflicting values - %s and %s'),
              $sourcepackage, $v);
    }
}

1;
