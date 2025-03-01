# -*- cperl -*-
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.
#
# Copyright (C) 2002-2014 Jens Thoms Toerring <jt@toerring.de>


# Package for file locking with fcntl(2) in which the binary layout of
# the C flock struct has been determined via a C program on installation
# and appropriate Perl code been appended to the package.

package File::FcntlLock::Pure;

use 5.006;
use strict;
use warnings;
use base qw( File::FcntlLock::Core );


our $VERSION = File::FcntlLock::Core->VERSION;

our @EXPORT = @File::FcntlLock::Core::EXPORT;


###########################################################
# Function for doing the actual fcntl() call: assembles the binary
# structure that must be passed to fcntl() from the File::FcntlLock
# object we get passed, calls it and then modifies the File::FcntlLock
# with the data from the flock structure

sub lock {
    my ( $self, $fh, $action ) = @_;
    my $buf = $self->pack_flock( );
    my $ret = fcntl( $fh, $action, $buf );

    if ( $ret  ) {
		$self->unpack_flock( $buf );
        $self->{ errno } = $self->{ error } = undef;
    } else {
        $self->get_error( $self->{ errno } = $! + 0 );
    }

    return $ret;
}


###########################################################

# Method created automatically while running 'perl Makefile.PL'
# (based on the the C 'struct flock' in <fcntl.h>) for packing
# the data from the 'flock_struct' into a binary blob to be
# passed to fcntl().

sub pack_flock {
    my $self = shift;
    return pack( 'ssx4qqlx4',
                 $self->{ l_type },
                 $self->{ l_whence },
                 $self->{ l_start },
                 $self->{ l_len },
                 $self->{ l_pid } );
}


###########################################################

# Method created automatically while running 'perl Makefile.PL'
# (based on the the C 'struct flock' in <fcntl.h>) for unpacking
# the binary blob received from a call of fcntl() into the
# 'flock_struct'.

sub unpack_flock {
     my ( $self, $data ) = @_;
     ( $self->{ l_type   },
       $self->{ l_whence },
       $self->{ l_start  },
       $self->{ l_len    },
       $self->{ l_pid    } ) = unpack( 'ssx4qqlx4', $data );
}


1;
