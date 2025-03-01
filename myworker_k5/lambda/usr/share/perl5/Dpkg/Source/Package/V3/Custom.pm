# Copyright © 2008 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::Source::Package::V3::Custom;

use strict;
use warnings;

our $VERSION = '0.01';

use Dpkg::Gettext;
use Dpkg::ErrorHandling;

use parent qw(Dpkg::Source::Package);

our $CURRENT_MINOR_VERSION = '0';

my @module_cmdline = (
    {
        name => '--target-format=<value>',
        help => N_('define the format of the generated source package'),
        when => 'build',
    }
);

sub describe_cmdline_options {
    return @module_cmdline;
}

sub parse_cmdline_option {
    my ($self, $opt) = @_;
    if ($opt =~ /^--target-format=(.*)$/) {
        $self->{options}{target_format} = $1;
        return 1;
    }
    return 0;
}
sub do_extract {
    error(g_("Format '3.0 (custom)' is only used to create source packages"));
}

sub can_build {
    my ($self, $dir) = @_;

    return (0, g_('no files indicated on command line'))
        unless scalar(@{$self->{options}{ARGV}});
    return 1;
}

sub do_build {
    my ($self, $dir) = @_;
    # Update real target format
    my $format = $self->{options}{target_format};
    error(g_('--target-format option is missing')) unless $format;
    $self->{fields}{'Format'} = $format;
    # Add all files
    foreach my $file (@{$self->{options}{ARGV}}) {
        $self->add_file($file);
    }
}

1;
