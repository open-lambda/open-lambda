# Copyright © 2008-2010 Raphaël Hertzog <hertzog@debian.org>
# Copyright © 2012-2014 Guillem Jover <guillem@debian.org>
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

package Dpkg::Compression::FileHandle;

use strict;
use warnings;

our $VERSION = '1.01';

use Carp;

use Dpkg::Compression;
use Dpkg::Compression::Process;
use Dpkg::Gettext;
use Dpkg::ErrorHandling;

use parent qw(IO::File Tie::Handle);

# Useful reference to understand some kludges required to
# have the class behave like a filehandle
# http://blog.woobling.org/2009/10/are-filehandles-objects.html

=encoding utf8

=head1 NAME

Dpkg::Compression::FileHandle - class dealing transparently with file compression

=head1 SYNOPSIS

    use Dpkg::Compression::FileHandle;

    my ($fh, @lines);

    $fh = Dpkg::Compression::FileHandle->new(filename => 'sample.gz');
    print $fh "Something\n";
    close $fh;

    $fh = Dpkg::Compression::FileHandle->new();
    open($fh, '>', 'sample.bz2');
    print $fh "Something\n";
    close $fh;

    $fh = Dpkg::Compression::FileHandle->new();
    $fh->open('sample.xz', 'w');
    $fh->print("Something\n");
    $fh->close();

    $fh = Dpkg::Compression::FileHandle->new(filename => 'sample.gz');
    @lines = <$fh>;
    close $fh;

    $fh = Dpkg::Compression::FileHandle->new();
    open($fh, '<', 'sample.bz2');
    @lines = <$fh>;
    close $fh;

    $fh = Dpkg::Compression::FileHandle->new();
    $fh->open('sample.xz', 'r');
    @lines = $fh->getlines();
    $fh->close();

=head1 DESCRIPTION

Dpkg::Compression::FileHandle is a class that can be used
like any filehandle and that deals transparently with compressed
files. By default, the compression scheme is guessed from the filename
but you can override this behaviour with the method C<set_compression>.

If you don't open the file explicitly, it will be auto-opened on the
first read or write operation based on the filename set at creation time
(or later with the C<set_filename> method).

Once a file has been opened, the filehandle must be closed before being
able to open another file.

=head1 STANDARD FUNCTIONS

The standard functions acting on filehandles should accept a
Dpkg::Compression::FileHandle object transparently including
C<open> (only when using the variant with 3 parameters), C<close>,
C<binmode>, C<eof>, C<fileno>, C<getc>, C<print>, C<printf>, C<read>,
C<sysread>, C<say>, C<write>, C<syswrite>, C<seek>, C<sysseek>, C<tell>.

Note however that C<seek> and C<sysseek> will only work on uncompressed
files as compressed files are really pipes to the compressor programs
and you can't seek on a pipe.

=head1 FileHandle METHODS

The class inherits from IO::File so all methods that work on this
class should work for Dpkg::Compression::FileHandle too. There
may be exceptions though.

=head1 PUBLIC METHODS

=over 4

=item $fh = Dpkg::Compression::FileHandle->new(%opts)

Creates a new filehandle supporting on-the-fly compression/decompression.
Supported options are "filename", "compression", "compression_level" (see
respective set_* functions) and "add_comp_ext". If "add_comp_ext"
evaluates to true, then the extension corresponding to the selected
compression scheme is automatically added to the recorded filename. It's
obviously incompatible with automatic detection of the compression method.

=cut

# Class methods
sub new {
    my ($this, %args) = @_;
    my $class = ref($this) || $this;
    my $self = IO::File->new();
    # Tying is required to overload the open functions and to auto-open
    # the file on first read/write operation
    tie *$self, $class, $self; ## no critic (Miscellanea::ProhibitTies)
    bless $self, $class;
    # Initializations
    *$self->{compression} = 'auto';
    *$self->{compressor} = Dpkg::Compression::Process->new();
    *$self->{add_comp_ext} = $args{add_compression_extension} ||
	    $args{add_comp_ext} || 0;
    *$self->{allow_sigpipe} = 0;
    if (exists $args{filename}) {
	$self->set_filename($args{filename});
    }
    if (exists $args{compression}) {
	$self->set_compression($args{compression});
    }
    if (exists $args{compression_level}) {
	$self->set_compression_level($args{compression_level});
    }
    return $self;
}

=item $fh->ensure_open($mode, %opts)

Ensure the file is opened in the requested mode ("r" for read and "w" for
write). The options are passed down to the compressor's spawn() call, if one
is used. Opens the file with the recorded filename if needed. If the file
is already open but not in the requested mode, then it errors out.

=cut

sub ensure_open {
    my ($self, $mode, %opts) = @_;
    if (exists *$self->{mode}) {
	return if *$self->{mode} eq $mode;
	croak "ensure_open requested incompatible mode: $mode";
    } else {
	# Sanitize options.
	delete $opts{from_pipe};
	delete $opts{from_file};
	delete $opts{to_pipe};
	delete $opts{to_file};

	if ($mode eq 'w') {
	    $self->_open_for_write(%opts);
	} elsif ($mode eq 'r') {
	    $self->_open_for_read(%opts);
	} else {
	    croak "invalid mode in ensure_open: $mode";
	}
    }
}

##
## METHODS FOR TIED HANDLE
##
sub TIEHANDLE {
    my ($class, $self) = @_;
    return $self;
}

sub WRITE {
    my ($self, $scalar, $length, $offset) = @_;
    $self->ensure_open('w');
    return *$self->{file}->write($scalar, $length, $offset);
}

sub READ {
    my ($self, $scalar, $length, $offset) = @_;
    $self->ensure_open('r');
    return *$self->{file}->read($scalar, $length, $offset);
}

sub READLINE {
    my ($self) = shift;
    $self->ensure_open('r');
    return *$self->{file}->getlines() if wantarray;
    return *$self->{file}->getline();
}

sub OPEN {
    my ($self) = shift;
    if (scalar(@_) == 2) {
	my ($mode, $filename) = @_;
	$self->set_filename($filename);
	if ($mode eq '>') {
	    $self->_open_for_write();
	} elsif ($mode eq '<') {
	    $self->_open_for_read();
	} else {
	    croak 'Dpkg::Compression::FileHandle does not support ' .
	          "open() mode $mode";
	}
    } else {
	croak 'Dpkg::Compression::FileHandle only supports open() ' .
	      'with 3 parameters';
    }
    return 1; # Always works (otherwise errors out)
}

sub CLOSE {
    my ($self) = shift;
    my $ret = 1;
    if (defined *$self->{file}) {
	$ret = *$self->{file}->close(@_) if *$self->{file}->opened();
    } else {
	$ret = 0;
    }
    $self->_cleanup();
    return $ret;
}

sub FILENO {
    my ($self) = shift;
    return *$self->{file}->fileno(@_) if defined *$self->{file};
    return;
}

sub EOF {
    # Since perl 5.12, an integer parameter is passed describing how the
    # function got called, just ignore it.
    my ($self, $param) = (shift, shift);
    return *$self->{file}->eof(@_) if defined *$self->{file};
    return 1;
}

sub SEEK {
    my ($self) = shift;
    return *$self->{file}->seek(@_) if defined *$self->{file};
    return 0;
}

sub TELL {
    my ($self) = shift;
    return *$self->{file}->tell(@_) if defined *$self->{file};
    return -1;
}

sub BINMODE {
    my ($self) = shift;
    return *$self->{file}->binmode(@_) if defined *$self->{file};
    return;
}

##
## NORMAL METHODS
##

=item $fh->set_compression($comp)

Defines the compression method used. $comp should one of the methods supported by
B<Dpkg::Compression> or "none" or "auto". "none" indicates that the file is
uncompressed and "auto" indicates that the method must be guessed based
on the filename extension used.

=cut

sub set_compression {
    my ($self, $method) = @_;
    if ($method ne 'none' and $method ne 'auto') {
	*$self->{compressor}->set_compression($method);
    }
    *$self->{compression} = $method;
}

=item $fh->set_compression_level($level)

Indicate the desired compression level. It should be a value accepted
by the function C<compression_is_valid_level> of B<Dpkg::Compression>.

=cut

sub set_compression_level {
    my ($self, $level) = @_;
    *$self->{compressor}->set_compression_level($level);
}

=item $fh->set_filename($name, [$add_comp_ext])

Use $name as filename when the file must be opened/created. If
$add_comp_ext is passed, it indicates whether the default extension
of the compression method must be automatically added to the filename
(or not).

=cut

sub set_filename {
    my ($self, $filename, $add_comp_ext) = @_;
    *$self->{filename} = $filename;
    # Automatically add compression extension to filename
    if (defined($add_comp_ext)) {
	*$self->{add_comp_ext} = $add_comp_ext;
    }
    my $comp_ext_regex = compression_get_file_extension_regex();
    if (*$self->{add_comp_ext} and $filename =~ /\.$comp_ext_regex$/) {
	warning('filename %s already has an extension of a compressed file ' .
	        'and add_comp_ext is active', $filename);
    }
}

=item $file = $fh->get_filename()

Returns the filename that would be used when the filehandle must
be opened (both in read and write mode). This function errors out
if "add_comp_ext" is enabled while the compression method is set
to "auto". The returned filename includes the extension of the compression
method if "add_comp_ext" is enabled.

=cut

sub get_filename {
    my $self = shift;
    my $comp = *$self->{compression};
    if (*$self->{add_comp_ext}) {
	if ($comp eq 'auto') {
	    croak 'automatic detection of compression is ' .
	          'incompatible with add_comp_ext';
	} elsif ($comp eq 'none') {
	    return *$self->{filename};
	} else {
	    return *$self->{filename} . '.' .
	           compression_get_property($comp, 'file_ext');
	}
    } else {
	return *$self->{filename};
    }
}

=item $ret = $fh->use_compression()

Returns "0" if no compression is used and the compression method used
otherwise. If the compression is set to "auto", the value returned
depends on the extension of the filename obtained with the B<get_filename>
method.

=cut

sub use_compression {
    my $self = shift;
    my $comp = *$self->{compression};
    if ($comp eq 'none') {
	return 0;
    } elsif ($comp eq 'auto') {
	$comp = compression_guess_from_filename($self->get_filename());
	*$self->{compressor}->set_compression($comp) if $comp;
    }
    return $comp;
}

=item $real_fh = $fh->get_filehandle()

Returns the real underlying filehandle. Useful if you want to pass it
along in a derived class.

=cut

sub get_filehandle {
    my $self = shift;
    return *$self->{file} if exists *$self->{file};
}

## INTERNAL METHODS

sub _open_for_write {
    my ($self, %opts) = @_;
    my $filehandle;

    croak 'cannot reopen an already opened compressed file'
        if exists *$self->{mode};

    if ($self->use_compression()) {
	*$self->{compressor}->compress(from_pipe => \$filehandle,
		to_file => $self->get_filename(), %opts);
    } else {
	CORE::open($filehandle, '>', $self->get_filename)
	    or syserr(g_('cannot write %s'), $self->get_filename());
    }
    *$self->{mode} = 'w';
    *$self->{file} = $filehandle;
}

sub _open_for_read {
    my ($self, %opts) = @_;
    my $filehandle;

    croak 'cannot reopen an already opened compressed file'
        if exists *$self->{mode};

    if ($self->use_compression()) {
	*$self->{compressor}->uncompress(to_pipe => \$filehandle,
		from_file => $self->get_filename(), %opts);
        *$self->{allow_sigpipe} = 1;
    } else {
	CORE::open($filehandle, '<', $self->get_filename)
	    or syserr(g_('cannot read %s'), $self->get_filename());
    }
    *$self->{mode} = 'r';
    *$self->{file} = $filehandle;
}

sub _cleanup {
    my $self = shift;
    my $cmdline = *$self->{compressor}{cmdline} // '';
    *$self->{compressor}->wait_end_process(nocheck => *$self->{allow_sigpipe});
    if (*$self->{allow_sigpipe}) {
        require POSIX;
        unless (($? == 0) || (POSIX::WIFSIGNALED($?) &&
                              (POSIX::WTERMSIG($?) == POSIX::SIGPIPE()))) {
            subprocerr($cmdline);
        }
	*$self->{allow_sigpipe} = 0;
    }
    delete *$self->{mode};
    delete *$self->{file};
}

=back

=head1 DERIVED CLASSES

If you want to create a class that inherits from
Dpkg::Compression::FileHandle you must be aware that
the object is a reference to a GLOB that is returned by Symbol::gensym()
and as such it's not a HASH.

You can store internal data in a hash but you have to use
C<*$self->{...}> to access the associated hash like in the example below:

    sub set_option {
	my ($self, $value) = @_;
	*$self->{option} = $value;
    }

=head1 CHANGES

=head2 Version 1.01 (dpkg 1.17.11)

New argument: $fh->ensure_open() accepts an %opts argument.

=head2 Version 1.00 (dpkg 1.15.6)

Mark the module as public.

=cut
1;
