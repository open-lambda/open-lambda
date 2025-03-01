# -*- cperl -*-
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.
#
# Copyright (C) 2002-2014 Jens Thoms Toerring <jt@toerring.de>


# Package for file locking with fcntl(2) in which the binary layout of
# the C flock struct is determined via compiling and running a C program
# each time the package is loaded

package File::FcntlLock::Inline;

use 5.006001;
use strict;
use warnings;
use Fcntl;
use Config;
use File::Temp;
use File::Spec;
use base qw( File::FcntlLock::Core );


our $VERSION = File::FcntlLock::Core->VERSION;

our @EXPORT = @File::FcntlLock::Core::EXPORT;


my ( $packstr, @member_list );


###########################################################

BEGIN {
	# Create a C file in the preferred directory for temporary files for
	# probing the layout of the C 'flock struct'. Since __DATA__ can't
	# be used in a BEGIN block we've got to do with a HEREDOC.

	my $c_file = File::Temp->new( TEMPLATE => 'File-FcntlLock-XXXXXX',
								  SUFFIX   => '.c',
								  DIR      => File::Spec->tmpdir( ) );

	print $c_file <<EOF;
#include <stdio.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
#include <limits.h>


#define membersize( type, member ) ( sizeof( ( ( type * ) NULL )->member ) )
#define NUM_ELEMS( p ) ( sizeof p / sizeof *p )

typedef struct {
    const char * name;
    size_t       size;
    size_t       offset;
}  Params;


/*-------------------------------------------------*
 * Called from qsort() for sorting an array of Params structures
 * in ascending order of their 'offset' members
 *-------------------------------------------------*/

static int
comp( const void * a,
      const void * b )
{
    if ( a == b )
        return 0;
    return ( ( Params * ) a )->offset < ( ( Params * ) b )->offset ? -1 : 1;
}


/*-------------------------------------------------*
 *-------------------------------------------------*/

int
main( void )
{
    Params params[ ] = { { "l_type",
                           CHAR_BIT * membersize( struct flock, l_type ),
                           CHAR_BIT * offsetof( struct flock, l_type ) },
                         { "l_whence",
                           CHAR_BIT * membersize( struct flock, l_whence ),
                           CHAR_BIT * offsetof( struct flock, l_whence ) },
                         { "l_start",
                           CHAR_BIT * membersize( struct flock, l_start ),
                           CHAR_BIT * offsetof( struct flock, l_start ) },
                         { "l_len",
                           CHAR_BIT * membersize( struct flock, l_len ),
                           CHAR_BIT * offsetof( struct flock, l_len ) },
                         { "l_pid",
                           CHAR_BIT * membersize( struct flock, l_pid ),
                           CHAR_BIT * offsetof( struct flock, l_pid ) } };
    size_t size = CHAR_BIT * sizeof( struct flock );
    size_t i;
    size_t pos = 0;
    char packstr[ 128 ] = "";
    
    /* All sizes and offsets must be divisable by 8 and the sizes of the
       members must be either 8-, 16-, 32- or 64-bit values, otherwise
       there's no good way to pack them. */

    if ( size % 8 )
        exit( EXIT_FAILURE );

    size /= 8;

    for ( i = 0; i < NUM_ELEMS( params ); ++i )
    {
        if (    params[ i ].size   % 8
             || params[ i ].offset % 8
             || (    params[ i ].size   != 8
                  && params[ i ].size   != 16
                  && params[ i ].size   != 32
                  && params[ i ].size   != 64 ) )
            exit( EXIT_FAILURE );

        params[ i ].size   /= 8;
        params[ i ].offset /= 8;
    }

    /* Sort the array of structures for the members in ascending order of
       the offset */

    qsort( params, NUM_ELEMS( params ), sizeof *params, comp );

    /* Cobble together the template string to be passed to pack(), taking
       care of padding and also extra members we're not interested in. All
       the interesting members have signed integer types. */

    for ( i = 0; i < NUM_ELEMS( params ); ++i )
    {
		if ( pos != params[ i ].offset )
			sprintf( packstr + strlen( packstr ), "x%lu",
					 ( unsigned long )( params[ i ].offset - pos ) );
		pos = params[ i ].offset;

        switch ( params[ i ].size )
        {
            case 1 :
				strcat( packstr, "c" );
                break;

            case 2 :
				strcat( packstr, "s" );
                break;

            case 4 :
				strcat( packstr, "l" );
                break;

            case 8 :
#if defined NO_Q_FORMAT
                exit( EXIT_FAILURE );
#endif
				strcat( packstr, "q" );
                break;

            default :
                exit( EXIT_FAILURE );
        }

		pos += params[ i ].size;
    }

    if ( pos < size )
        sprintf( packstr + strlen( packstr ), "x%lu",
                 (unsigned long ) ( size - pos ) );

    printf( "%s\\n", packstr );
    for ( i = 0; i < NUM_ELEMS( params ); ++i )
		printf( "%s\\n", params[ i ].name );

    return 0;
}
EOF

	close $c_file;

	# Try to compile and link the file.

	my $exec_file = File::Temp->new( TEMPLATE => 'File=FcntlLock-XXXXXX',
									 DIR      => File::Spec->tmpdir( ) );
	close $exec_file;

    my $qflag = eval { pack 'q', 1 };
    $qflag = $@ ? '-DNO_Q_FORMAT' : '';

	die "Failed to run the C compiler '$Config{cc}'\n"
		if system "$Config{cc} $Config{ccflags} $qflag -o $exec_file $c_file";

	# Run the program and read it's output, it writes out the template string
	# we need for packing and unpacking the binary C struct flock required for
	# fcntl() and then the members of the structures in the sequence they are
	# defined in there.

	open my $pipe, '-|', $exec_file
		or die "Failed to run a compiled program: $!\n";

	chomp( $packstr = <$pipe> );
	while ( my $line = <$pipe> ) {
		chomp $line;
		push @member_list, $line;
	}

	# Make sure we got all information needed

    die   "Your Perl version does not support the 'q' format for pack() "
        . "and unpack()\n" unless defined $packstr;

	die "Failed to obtain all needed data about the C struct flock\n"
		unless @member_list == 5;
}


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
# Method for packing the data from the 'flock_struct' into a
# binary blob to be passed to fcntl().

sub pack_flock {
    my $self = shift;
	my @args;
	push @args, $self->{ $_ } for @member_list;
    return pack $packstr, @args;
}


###########################################################
# Method for unpacking the binary blob received from a call of
# fcntl() into the 'flock_struct'.

sub unpack_flock {
	my ( $self, $data ) = @_;
	my @res = unpack $packstr, $data;
	$self->{ $_ } = shift @res for @member_list;
}


=cut


1;


# Local variables:
# tab-width: 4
# indent-tabs-mode: nil
# End:
