# -*- cperl -*-
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.
#
# Copyright (C) 2002-2014 Jens Thoms Toerring <jt@toerring.de>


# Base class for the three modules for file locking using fcntl(2)

package File::FcntlLock::Core;

use 5.006;
use strict;
use warnings;
use POSIX;
use Carp;
use base qw( File::FcntlLock::Errors Exporter );


our $VERSION = '0.22';


# Items to export into callers namespace by default.

our @EXPORT = qw( F_GETLK F_SETLK F_SETLKW
                  F_RDLCK F_WRLCK F_UNLCK
                  SEEK_SET SEEK_CUR SEEK_END );


###########################################################
#
# Make our exports exportable by child classes

sub import
{
    File::FcntlLock::Core->export_to_level( 1, @_ );
}


###########################################################
# Method for creating the object

sub new {
    my $inv = shift;
    my $pkg = ref( $inv ) || $inv;

    my $self = { l_type        => F_RDLCK,
                 l_whence      => SEEK_SET,
                 l_start       => 0,
                 l_len         => 0,
                 l_pid         => 0,
                 errno         => undef,
                 error_message => undef      };

    if ( @_ % 2 ) {
        carp "Missing value in key-value initializer list " .
             "in call of new method";
        return;
    }

    while ( @_ ) {
        my $key = shift;
        no strict 'refs';
        unless ( defined &$key ) {
            carp "Flock structure has no '$key' member " .
                 "in call of new method";
            return;
        }
        &$key( $self, shift );
        use strict 'refs';
    }

    bless $self, $pkg;
}


###########################################################
# Method for setting or querying the 'l_type' property

sub l_type {
    my $self = shift;

    if ( @_ ) {
        my $l_type = shift;
        unless (    $l_type == F_RDLCK
                 or $l_type == F_WRLCK
                 or $l_type == F_UNLCK ) {
            carp "Invalid argument in call of l_type method";
            return;
        }
        $self->{ l_type } = $l_type;
    }
    return $self->{ l_type };
}


###########################################################
# Method for setting or querying the 'l_whence' property

sub l_whence {
    my $self = shift;

    if ( @_ ) {
        my $l_whence = shift;
        unless (    $l_whence == SEEK_SET
                 or $l_whence == SEEK_CUR
                 or $l_whence == SEEK_END ) {
            carp "Invalid argument in call of l_whence method";
            return;
        }
        $self->{ l_whence } = $l_whence;
    }
    return $self->{ l_whence };
}


###########################################################
# Method to set or query of the 'l_start' property

sub l_start {
    my $self = shift;

    $self->{ l_start } = shift if @_;
    return $self->{ l_start };
}


###########################################################
# Method to set or query the 'l_len' property

sub l_len {
    my $self = shift;

    $self->{ l_len } = shift if @_;
    return $self->{ l_len };
}


###########################################################
# Method to query the 'l_pid' property

sub l_pid {
    return shift->{ l_pid };
}


1;


# Local variables:
# tab-width: 4
# indent-tabs-mode: nil
# End:
