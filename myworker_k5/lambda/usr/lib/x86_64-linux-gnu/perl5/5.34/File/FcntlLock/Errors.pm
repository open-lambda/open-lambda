# -*- cperl -*-
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.
#
# Copyright (C) 2002-2014 Jens Thoms Toerring <jt@toerring.de>


# Helper package for File::FcntLock::Core for handling error messages

package File::FcntlLock::Errors;


use 5.006;
use strict;
use warnings;
use Errno;


my %fcntl_error_texts;


BEGIN {
    # Set up a hash with the error messages, but only for errno's that Errno
    # knows about. The texts represent what is written in SUSv3 and in the
    # man pages for Linux, TRUE64, OpenBSD3 and Solaris8.

    my $err;

    if ( $err = eval { &Errno::EACCES } ) {
        $fcntl_error_texts{ $err } = "File or segment already locked " .
                                     "by other process(es) or file is " .
                                     "mmap()ed to virtual memory";
    }

    if ( $err = eval { &Errno::EAGAIN } ) {
        $fcntl_error_texts{ $err } = "File or segment already locked " .
                                     "by other process(es)";
    }

    if ( $err = eval { &Errno::EBADF } ) {
        $fcntl_error_texts{ $err } = "Not an open file or not opened for " .
                                     "writing (with F_WRLCK) or reading " .
                                     "(with F_RDLCK)";
    }

    if ( $err = eval { &Errno::EDEADLK } ) {
        $fcntl_error_texts{ $err } = "Operation would cause a deadlock";
    }

    if ( $err = eval { &Errno::EFAULT } ) {
        $fcntl_error_texts{ $err } = "Lock outside accessible address space " .
                                     "or to many locked regions";
    }

    if ( $err = eval { &Errno::EINTR } ) {
        $fcntl_error_texts{ $err } = "Operation interrupted by a signal";
    }

    if ( $err = eval { &Errno::ENOLCK } ) {
        $fcntl_error_texts{ $err } = "Too many segment locks open, lock " .
                                     "table full or remote locking protocol " .
                                     "failure (e.g. NFS)";
    }

    if ( $err = eval { &Errno::EINVAL } ) {
        $fcntl_error_texts{ $err } = "Illegal parameter or file does not " .
                                     "support locking";
    }

    if ( $err = eval { &Errno::EOVERFLOW } ) {
        $fcntl_error_texts{ $err } = "One of the parameters to be returned " .
                                     "can not be represented correctly";
    }

    if ( $err = eval { &Errno::ENETUNREACH } ) {
        $fcntl_error_texts{ $err } = "File is on remote machine that can " .
                                     "not be reached anymore";
    }

    if ( $err = eval { &Errno::ENOLINK } ) {
        $fcntl_error_texts{ $err } = "File is on remote machine that can " .
                                     "not be reached anymore";
    }
}


###########################################################
# Function for converting an errno to a useful, human readable
# message.

sub get_error {
    my ( $self, $err ) = @_;
    return $self->{ error } =
             defined $fcntl_error_texts{ $err } ? $fcntl_error_texts{ $err }
                                                : "Unexpected error: $!";
}


###########################################################
# Method returns the error number from the latest call of the
# derived classes lock() function. If the last call did not
# result in an error the method returns undef.

sub lock_errno {
    return shift->{ errno };
}


###########################################################
# Method returns a short description of the error that happenend
# on the latest call of derived classes lock() method with the
# object. If there was no error the method returns undef.

sub error {
    return shift->{ error };
}


###########################################################
# Method returns the "normal" system error message associated
# with errno. The method returns undef if there was no error.

sub system_error {
    local $!;
    my $self = shift;
    return $self->{ errno } ? $! = $self->{ errno } : undef;
}


1;


# Local variables:
# tab-width: 4
# indent-tabs-mode: nil
# End:
