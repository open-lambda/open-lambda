# Copyright © 2009-2010 Raphaël Hertzog <hertzog@debian.org>
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

package Dpkg::Conf;

use strict;
use warnings;

our $VERSION = '1.04';

use Carp;

use Dpkg::Gettext;
use Dpkg::ErrorHandling;

use parent qw(Dpkg::Interface::Storable);

use overload
    '@{}' => sub { return [ $_[0]->get_options() ] },
    fallback => 1;

=encoding utf8

=head1 NAME

Dpkg::Conf - parse dpkg configuration files

=head1 DESCRIPTION

The Dpkg::Conf object can be used to read options from a configuration
file. It can export an array that can then be parsed exactly like @ARGV.

=head1 METHODS

=over 4

=item $conf = Dpkg::Conf->new(%opts)

Create a new Dpkg::Conf object. Some options can be set through %opts:
if allow_short evaluates to true (it defaults to false), then short
options are allowed in the configuration file, they should be prepended
with a single hyphen.

=cut

sub new {
    my ($this, %opts) = @_;
    my $class = ref($this) || $this;

    my $self = {
	options => [],
	allow_short => 0,
    };
    foreach my $opt (keys %opts) {
	$self->{$opt} = $opts{$opt};
    }
    bless $self, $class;

    return $self;
}

=item @$conf

=item @options = $conf->get_options()

Returns the list of options that can be parsed like @ARGV.

=cut

sub get_options {
    my $self = shift;

    return @{$self->{options}};
}

=item $conf->load($file)

Read options from a file. Return the number of options parsed.

=item $conf->load_system_config($file)

Read options from a system configuration file.

Return the number of options parsed.

=cut

sub load_system_config {
    my ($self, $file) = @_;

    return 0 unless -e "$Dpkg::CONFDIR/$file";
    return $self->load("$Dpkg::CONFDIR/$file");
}

=item $conf->load_user_config($file)

Read options from a user configuration file. It will try to use the XDG
directory, either $XDG_CONFIG_HOME/dpkg/ or $HOME/.config/dpkg/.

Return the number of options parsed.

=cut

sub load_user_config {
    my ($self, $file) = @_;

    my $confdir = $ENV{XDG_CONFIG_HOME};
    $confdir ||= $ENV{HOME} . '/.config' if length $ENV{HOME};

    return 0 unless length $confdir;
    return 0 unless -e "$confdir/dpkg/$file";
    return $self->load("$confdir/dpkg/$file") if length $confdir;
    return 0;
}

=item $conf->load_config($file)

Read options from system and user configuration files.

Return the number of options parsed.

=cut

sub load_config {
    my ($self, $file) = @_;

    my $nopts = 0;

    $nopts += $self->load_system_config($file);
    $nopts += $self->load_user_config($file);

    return $nopts;
}

=item $conf->parse($fh)

Parse options from a file handle. When called multiple times, the parsed
options are accumulated.

Return the number of options parsed.

=cut

sub parse {
    my ($self, $fh, $desc) = @_;
    my $count = 0;
    local $_;

    while (<$fh>) {
	chomp;
	s/^\s+//;             # Strip leading spaces
	s/\s+$//;             # Strip trailing spaces
	s/\s+=\s+/=/;         # Remove spaces around the first =
	s/\s+/=/ unless m/=/; # First spaces becomes = if no =
	# Skip empty lines and comments
	next if /^#/ or length == 0;
	if (/^-[^-]/ and not $self->{allow_short}) {
	    warning(g_('short option not allowed in %s, line %d'), $desc, $.);
	    next;
	}
	if (/^([^=]+)(?:=(.*))?$/) {
	    my ($name, $value) = ($1, $2);
	    $name = "--$name" unless $name =~ /^-/;
	    if (defined $value) {
		$value =~ s/^"(.*)"$/$1/ or $value =~ s/^'(.*)'$/$1/;
		push @{$self->{options}}, "$name=$value";
	    } else {
		push @{$self->{options}}, $name;
	    }
	    $count++;
	} else {
	    warning(g_('invalid syntax for option in %s, line %d'), $desc, $.);
	}
    }
    return $count;
}

=item $conf->filter(%opts)

Filter the list of options, either removing or keeping all those that
return true when $opts{remove}->($option) or $opts{keep}->($option) is called.

=cut

sub filter {
    my ($self, %opts) = @_;
    my $remove = $opts{remove} // sub { 0 };
    my $keep = $opts{keep} // sub { 1 };

    @{$self->{options}} = grep { not $remove->($_) and $keep->($_) }
                               @{$self->{options}};
}

=item $string = $conf->output([$fh])

Write the options in the given filehandle (if defined) and return a string
representation of the content (that would be) written.

=item "$conf"

Return a string representation of the content.

=cut

sub output {
    my ($self, $fh) = @_;
    my $ret = '';
    foreach my $opt ($self->get_options()) {
	$opt =~ s/^--//;
	$opt =~ s/^([^=]+)=(.*)$/$1 = "$2"/;
	$opt .= "\n";
	print { $fh } $opt if defined $fh;
	$ret .= $opt;
    }
    return $ret;
}

=item $conf->save($file)

Save the options in a file.

=back

=head1 CHANGES

=head2 Version 1.04 (dpkg 1.20.0)

Remove croak: For 'format_argv' in $conf->filter().

Remove methods: $conf->get(), $conf->set().

=head2 Version 1.03 (dpkg 1.18.8)

Obsolete option: 'format_argv' in $conf->filter().

Obsolete methods: $conf->get(), $conf->set().

New methods: $conf->load_system_config(), $conf->load_system_user(),
$conf->load_config().

=head2 Version 1.02 (dpkg 1.18.5)

New option: Accept new option 'format_argv' in $conf->filter().

New methods: $conf->get(), $conf->set().

=head2 Version 1.01 (dpkg 1.15.8)

New method: $conf->filter()

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut

1;
