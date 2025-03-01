# Copyright © 2010-2011 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::BuildFlags;

use strict;
use warnings;

our $VERSION = '1.04';

use Dpkg ();
use Dpkg::Gettext;
use Dpkg::Build::Env;
use Dpkg::ErrorHandling;
use Dpkg::Vendor qw(run_vendor_hook);

=encoding utf8

=head1 NAME

Dpkg::BuildFlags - query build flags

=head1 DESCRIPTION

This class is used by dpkg-buildflags and can be used
to query the same information.

=head1 METHODS

=over 4

=item $bf = Dpkg::BuildFlags->new()

Create a new Dpkg::BuildFlags object. It will be initialized based
on the value of several configuration files and environment variables.

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;

    my $self = {
    };
    bless $self, $class;
    $self->load_vendor_defaults();
    return $self;
}

=item $bf->load_vendor_defaults()

Reset the flags stored to the default set provided by the vendor.

=cut

sub load_vendor_defaults {
    my $self = shift;

    $self->{features} = {};
    $self->{flags} = {
	ASFLAGS => '',
	CPPFLAGS => '',
	CFLAGS   => '',
	CXXFLAGS => '',
	OBJCFLAGS   => '',
	OBJCXXFLAGS => '',
	GCJFLAGS => '',
	DFLAGS   => '',
	FFLAGS   => '',
	FCFLAGS  => '',
	LDFLAGS  => '',
    };
    $self->{origin} = {
	ASFLAGS => 'vendor',
	CPPFLAGS => 'vendor',
	CFLAGS   => 'vendor',
	CXXFLAGS => 'vendor',
	OBJCFLAGS   => 'vendor',
	OBJCXXFLAGS => 'vendor',
	GCJFLAGS => 'vendor',
	DFLAGS   => 'vendor',
	FFLAGS   => 'vendor',
	FCFLAGS  => 'vendor',
	LDFLAGS  => 'vendor',
    };
    $self->{maintainer} = {
	ASFLAGS => 0,
	CPPFLAGS => 0,
	CFLAGS   => 0,
	CXXFLAGS => 0,
	OBJCFLAGS   => 0,
	OBJCXXFLAGS => 0,
	GCJFLAGS => 0,
	DFLAGS   => 0,
	FFLAGS   => 0,
	FCFLAGS  => 0,
	LDFLAGS  => 0,
    };
    # The vendor hook will add the feature areas build flags.
    run_vendor_hook('update-buildflags', $self);
}

=item $bf->load_system_config()

Update flags from the system configuration.

=cut

sub load_system_config {
    my $self = shift;

    $self->update_from_conffile("$Dpkg::CONFDIR/buildflags.conf", 'system');
}

=item $bf->load_user_config()

Update flags from the user configuration.

=cut

sub load_user_config {
    my $self = shift;

    my $confdir = $ENV{XDG_CONFIG_HOME};
    $confdir ||= $ENV{HOME} . '/.config' if length $ENV{HOME};
    if (length $confdir) {
        $self->update_from_conffile("$confdir/dpkg/buildflags.conf", 'user');
    }
}

=item $bf->load_environment_config()

Update flags based on user directives stored in the environment. See
dpkg-buildflags(1) for details.

=cut

sub load_environment_config {
    my $self = shift;

    foreach my $flag (keys %{$self->{flags}}) {
	my $envvar = 'DEB_' . $flag . '_SET';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->set($flag, Dpkg::Build::Env::get($envvar), 'env');
	}
	$envvar = 'DEB_' . $flag . '_STRIP';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->strip($flag, Dpkg::Build::Env::get($envvar), 'env');
	}
	$envvar = 'DEB_' . $flag . '_APPEND';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->append($flag, Dpkg::Build::Env::get($envvar), 'env');
	}
	$envvar = 'DEB_' . $flag . '_PREPEND';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->prepend($flag, Dpkg::Build::Env::get($envvar), 'env');
	}
    }
}

=item $bf->load_maintainer_config()

Update flags based on maintainer directives stored in the environment. See
dpkg-buildflags(1) for details.

=cut

sub load_maintainer_config {
    my $self = shift;

    foreach my $flag (keys %{$self->{flags}}) {
	my $envvar = 'DEB_' . $flag . '_MAINT_SET';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->set($flag, Dpkg::Build::Env::get($envvar), undef, 1);
	}
	$envvar = 'DEB_' . $flag . '_MAINT_STRIP';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->strip($flag, Dpkg::Build::Env::get($envvar), undef, 1);
	}
	$envvar = 'DEB_' . $flag . '_MAINT_APPEND';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->append($flag, Dpkg::Build::Env::get($envvar), undef, 1);
	}
	$envvar = 'DEB_' . $flag . '_MAINT_PREPEND';
	if (Dpkg::Build::Env::has($envvar)) {
	    $self->prepend($flag, Dpkg::Build::Env::get($envvar), undef, 1);
	}
    }
}


=item $bf->load_config()

Call successively load_system_config(), load_user_config(),
load_environment_config() and load_maintainer_config() to update the
default build flags defined by the vendor.

=cut

sub load_config {
    my $self = shift;

    $self->load_system_config();
    $self->load_user_config();
    $self->load_environment_config();
    $self->load_maintainer_config();
}

=item $bf->unset($flag)

Unset the build flag $flag, so that it will not be known anymore.

=cut

sub unset {
    my ($self, $flag) = @_;

    delete $self->{flags}->{$flag};
    delete $self->{origin}->{$flag};
    delete $self->{maintainer}->{$flag};
}

=item $bf->set($flag, $value, $source, $maint)

Update the build flag $flag with value $value and record its origin as
$source (if defined). Record it as maintainer modified if $maint is
defined and true.

=cut

sub set {
    my ($self, $flag, $value, $src, $maint) = @_;
    $self->{flags}->{$flag} = $value;
    $self->{origin}->{$flag} = $src if defined $src;
    $self->{maintainer}->{$flag} = $maint if $maint;
}

=item $bf->set_feature($area, $feature, $enabled)

Update the boolean state of whether a specific feature within a known
feature area has been enabled. The only currently known feature areas
are "future", "qa", "sanitize", "hardening" and "reproducible".

=cut

sub set_feature {
    my ($self, $area, $feature, $enabled) = @_;
    $self->{features}{$area}{$feature} = $enabled;
}

=item $bf->strip($flag, $value, $source, $maint)

Update the build flag $flag by stripping the flags listed in $value and
record its origin as $source (if defined). Record it as maintainer modified
if $maint is defined and true.

=cut

sub strip {
    my ($self, $flag, $value, $src, $maint) = @_;
    foreach my $tostrip (split(/\s+/, $value)) {
	next unless length $tostrip;
	$self->{flags}->{$flag} =~ s/(^|\s+)\Q$tostrip\E(\s+|$)/ /g;
    }
    $self->{flags}->{$flag} =~ s/^\s+//g;
    $self->{flags}->{$flag} =~ s/\s+$//g;
    $self->{origin}->{$flag} = $src if defined $src;
    $self->{maintainer}->{$flag} = $maint if $maint;
}

=item $bf->append($flag, $value, $source, $maint)

Append the options listed in $value to the current value of the flag $flag.
Record its origin as $source (if defined). Record it as maintainer modified
if $maint is defined and true.

=cut

sub append {
    my ($self, $flag, $value, $src, $maint) = @_;
    if (length($self->{flags}->{$flag})) {
        $self->{flags}->{$flag} .= " $value";
    } else {
        $self->{flags}->{$flag} = $value;
    }
    $self->{origin}->{$flag} = $src if defined $src;
    $self->{maintainer}->{$flag} = $maint if $maint;
}

=item $bf->prepend($flag, $value, $source, $maint)

Prepend the options listed in $value to the current value of the flag $flag.
Record its origin as $source (if defined). Record it as maintainer modified
if $maint is defined and true.

=cut

sub prepend {
    my ($self, $flag, $value, $src, $maint) = @_;
    if (length($self->{flags}->{$flag})) {
        $self->{flags}->{$flag} = "$value " . $self->{flags}->{$flag};
    } else {
        $self->{flags}->{$flag} = $value;
    }
    $self->{origin}->{$flag} = $src if defined $src;
    $self->{maintainer}->{$flag} = $maint if $maint;
}


=item $bf->update_from_conffile($file, $source)

Update the current build flags based on the configuration directives
contained in $file. See dpkg-buildflags(1) for the format of the directives.

$source is the origin recorded for any build flag set or modified.

=cut

sub update_from_conffile {
    my ($self, $file, $src) = @_;
    local $_;

    return unless -e $file;
    open(my $conf_fh, '<', $file) or syserr(g_('cannot read %s'), $file);
    while (<$conf_fh>) {
        chomp;
        next if /^\s*#/; # Skip comments
        next if /^\s*$/; # Skip empty lines
        if (/^(append|prepend|set|strip)\s+(\S+)\s+(\S.*\S)\s*$/i) {
            my ($op, $flag, $value) = ($1, $2, $3);
            unless (exists $self->{flags}->{$flag}) {
                warning(g_('line %d of %s mentions unknown flag %s'), $., $file, $flag);
                $self->{flags}->{$flag} = '';
            }
            if (lc($op) eq 'set') {
                $self->set($flag, $value, $src);
            } elsif (lc($op) eq 'strip') {
                $self->strip($flag, $value, $src);
            } elsif (lc($op) eq 'append') {
                $self->append($flag, $value, $src);
            } elsif (lc($op) eq 'prepend') {
                $self->prepend($flag, $value, $src);
            }
        } else {
            warning(g_('line %d of %s is invalid, it has been ignored'), $., $file);
        }
    }
    close($conf_fh);
}

=item $bf->get($flag)

Return the value associated to the flag. It might be undef if the
flag doesn't exist.

=cut

sub get {
    my ($self, $key) = @_;
    return $self->{flags}{$key};
}

=item $bf->get_feature_areas()

Return the feature areas (i.e. the area values has_features will return
true for).

=cut

sub get_feature_areas {
    my $self = shift;

    return keys %{$self->{features}};
}

=item $bf->get_features($area)

Return, for the given area, a hash with keys as feature names, and values
as booleans indicating whether the feature is enabled or not.

=cut

sub get_features {
    my ($self, $area) = @_;
    return %{$self->{features}{$area}};
}

=item $bf->get_origin($flag)

Return the origin associated to the flag. It might be undef if the
flag doesn't exist.

=cut

sub get_origin {
    my ($self, $key) = @_;
    return $self->{origin}{$key};
}

=item $bf->is_maintainer_modified($flag)

Return true if the flag is modified by the maintainer.

=cut

sub is_maintainer_modified {
    my ($self, $key) = @_;
    return $self->{maintainer}{$key};
}

=item $bf->has_features($area)

Returns true if the given area of features is known, and false otherwise.
The only currently recognized feature areas are "future", "qa", "sanitize",
"hardening" and "reproducible".

=cut

sub has_features {
    my ($self, $area) = @_;
    return exists $self->{features}{$area};
}

=item $bf->has($option)

Returns a boolean indicating whether the flags exists in the object.

=cut

sub has {
    my ($self, $key) = @_;
    return exists $self->{flags}{$key};
}

=item @flags = $bf->list()

Returns the list of flags stored in the object.

=cut

sub list {
    my $self = shift;
    my @list = sort keys %{$self->{flags}};
    return @list;
}

=back

=head1 CHANGES

=head2 Version 1.04 (dpkg 1.20.0)

New method: $bf->unset().

=head2 Version 1.03 (dpkg 1.16.5)

New method: $bf->get_feature_areas() to list possible values for
$bf->get_features.

New method $bf->is_maintainer_modified() and new optional parameter to
$bf->set(), $bf->append(), $bf->prepend(), $bf->strip().

=head2 Version 1.02 (dpkg 1.16.2)

New methods: $bf->get_features(), $bf->has_features(), $bf->set_feature().

=head2 Version 1.01 (dpkg 1.16.1)

New method: $bf->prepend() very similar to append(). Implement support of
the prepend operation everywhere.

New method: $bf->load_maintainer_config() that update the build flags
based on the package maintainer directives.

=head2 Version 1.00 (dpkg 1.15.7)

Mark the module as public.

=cut

1;
