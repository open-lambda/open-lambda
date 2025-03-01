# Copyright © 2007 Frank Lichtenheld <djpig@debian.org>
# Copyright © 2008, 2012-2017 Guillem Jover <guillem@debian.org>
# Copyright © 2010 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::BuildOptions;

use strict;
use warnings;

our $VERSION = '1.02';

use Dpkg::Gettext;
use Dpkg::ErrorHandling;
use Dpkg::Build::Env;

=encoding utf8

=head1 NAME

Dpkg::BuildOptions - parse and update build options

=head1 DESCRIPTION

This class can be used to manipulate options stored
in environment variables like DEB_BUILD_OPTIONS and
DEB_BUILD_MAINT_OPTIONS.

=head1 METHODS

=over 4

=item $bo = Dpkg::BuildOptions->new(%opts)

Create a new Dpkg::BuildOptions object. It will be initialized based
on the value of the environment variable named $opts{envvar} (or
DEB_BUILD_OPTIONS if that option is not set).

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;

    my $self = {
        options => {},
	source => {},
	envvar => $opts{envvar} // 'DEB_BUILD_OPTIONS',
    };
    bless $self, $class;
    $self->merge(Dpkg::Build::Env::get($self->{envvar}), $self->{envvar});
    return $self;
}

=item $bo->reset()

Reset the object to not have any option (it's empty).

=cut

sub reset {
    my $self = shift;
    $self->{options} = {};
    $self->{source} = {};
}

=item $bo->merge($content, $source)

Merge the options set in $content and record that they come from the
source $source. $source is mainly used in warning messages currently
to indicate where invalid options have been detected.

$content is a space separated list of options with optional assigned
values like "nocheck parallel=2".

=cut

sub merge {
    my ($self, $content, $source) = @_;
    return 0 unless defined $content;
    my $count = 0;
    foreach (split(/\s+/, $content)) {
	unless (/^([a-z][a-z0-9_-]*)(?:=(\S*))?$/) {
            warning(g_('invalid flag in %s: %s'), $source, $_);
            next;
        }
	$count += $self->set($1, $2, $source);
    }
    return $count;
}

=item $bo->set($option, $value, [$source])

Store the given option in the object with the given value. It's legitimate
for a value to be undefined if the option is a simple boolean (its
presence means true, its absence means false). The $source is optional
and indicates where the option comes from.

The known options have their values checked for sanity. Options without
values have their value removed and options with invalid values are
discarded.

=cut

sub set {
    my ($self, $key, $value, $source) = @_;

    # Sanity checks
    if ($key =~ /^(terse|noopt|nostrip|nocheck)$/ && defined($value)) {
	$value = undef;
    } elsif ($key eq 'parallel')  {
	$value //= '';
	return 0 if $value !~ /^\d*$/;
    }

    $self->{options}{$key} = $value;
    $self->{source}{$key} = $source;

    return 1;
}

=item $bo->get($option)

Return the value associated to the option. It might be undef even if the
option exists. You might want to check with $bo->has($option) to verify if
the option is stored in the object.

=cut

sub get {
    my ($self, $key) = @_;
    return $self->{options}{$key};
}

=item $bo->has($option)

Returns a boolean indicating whether the option is stored in the object.

=cut

sub has {
    my ($self, $key) = @_;
    return exists $self->{options}{$key};
}

=item $bo->parse_features($option, $use_feature)

Parse the $option values, as a set of known features to enable or disable,
as specified in the $use_feature hash reference.

Each feature is prefixed with a ‘B<+>’ or a ‘B<->’ character as a marker
to enable or disable it. The special feature “B<all>” can be used to act
on all known features.

Unknown or malformed features will emit warnings.

=cut

sub parse_features {
    my ($self, $option, $use_feature) = @_;

    foreach my $feature (split(/,/, $self->get($option) // '')) {
        $feature = lc $feature;
        if ($feature =~ s/^([+-])//) {
            my $value = ($1 eq '+') ? 1 : 0;
            if ($feature eq 'all') {
                $use_feature->{$_} = $value foreach keys %{$use_feature};
            } else {
                if (exists $use_feature->{$feature}) {
                    $use_feature->{$feature} = $value;
                } else {
                    warning(g_('unknown %s feature in %s variable: %s'),
                            $option, $self->{envvar}, $feature);
                }
            }
        } else {
            warning(g_('incorrect value in %s option of %s variable: %s'),
                    $option, $self->{envvar}, $feature);
        }
    }
}

=item $string = $bo->output($fh)

Return a string representation of the build options suitable to be
assigned to an environment variable. Can optionally output that string to
the given filehandle.

=cut

sub output {
    my ($self, $fh) = @_;
    my $o = $self->{options};
    my $res = join(' ', map { defined($o->{$_}) ? $_ . '=' . $o->{$_} : $_ } sort keys %$o);
    print { $fh } $res if defined $fh;
    return $res;
}

=item $bo->export([$var])

Export the build options to the given environment variable. If omitted,
the environment variable defined at creation time is assumed. The value
set to the variable is also returned.

=cut

sub export {
    my ($self, $var) = @_;
    $var //= $self->{envvar};
    my $content = $self->output();
    Dpkg::Build::Env::set($var, $content);
    return $content;
}

=back

=head1 CHANGES

=head2 Version 1.02 (dpkg 1.18.19)

New method: $bo->parse_features().

=head2 Version 1.01 (dpkg 1.16.1)

Enable to use another environment variable instead of DEB_BUILD_OPTIONS.
Thus add support for the "envvar" option at creation time.

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
