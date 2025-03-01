# -*- cperl -*-
#
# This program is free software; you can redistribute it and/or
# modify it under the same terms as Perl itself.
#
# Copyright (C) 2002-2014 Jens Thoms Toerring <jt@toerring.de>


# Alias for package for File::FcntlLock

package File::FcntlLock::XS;

use v5.6.1;
use strict;
use warnings;
use base qw( File::FcntlLock );

our $VERSION = File::FcntlLock->VERSION;

our @EXPORT = @File::FcntlLock::EXPORT;


1;


# Local variables:
# tab-width: 4
# indent-tabs-mode: nil
# End:
