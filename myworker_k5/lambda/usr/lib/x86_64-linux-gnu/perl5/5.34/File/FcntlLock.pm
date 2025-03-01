# -*- cperl -*-
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.
#
# Copyright (C) 2002-2014 Jens Thoms Toerring <jt@toerring.de>


package File::FcntlLock;

use 5.006;
use strict;
use warnings;
use POSIX;
use Errno;
use Carp;
use base qw( File::FcntlLock::Core DynaLoader );


our $VERSION = '0.22';


bootstrap File::FcntlLock $VERSION;

our @EXPORT = @File::FcntlLock::Core::EXPORT;


###########################################################
#
# Make our exports exportable by child classes

sub import
{
    File::FcntlLock->export_to_level( 1, @_ );
}


###########################################################
#
# Function for locking or unlocking a file or determining which
# process holds a lock.

sub lock {
    my ( $self, $fh, $action ) = @_;
    my ( $ret, $err );

    # Figure out the file descriptor - we might get a file handle, a
    # typeglob or already a file descriptor) and set it to a value which
    # will make fcntl(2) fail with EBADF if the argument is undefined or
    # is a file handle that's invalid.

    my $fd = ( ref( $fh ) or $fh =~ /^\*/ ) ? fileno( $fh ) : $fh;
    $fd = -1 unless defined $fd;

    # Set the action argument to something invalid if it's not defined
    # which then fcntl(2) fails and errno gets set accordingly

    $action = -1 unless defined $action;

    if ( $ret = C_fcntl_lock( $fd, $action, $self, $err ) ) {
        $self->{ errno } = $self->{ error } = undef;
    } elsif ( $err ) {
        die "Internal error in File::FcntlLock module detected";
    } else {
        $self->get_error( $self->{ errno } = $! + 0 );
    }

    return $ret;
}


1;


# Local variables:
# tab-width: 4
# indent-tabs-mode: nil
# End:
