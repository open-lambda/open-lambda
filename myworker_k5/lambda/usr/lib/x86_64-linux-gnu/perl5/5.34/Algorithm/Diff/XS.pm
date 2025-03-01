package Algorithm::Diff::XS;
use 5.006;
use strict;
use warnings;
use vars '$VERSION';
use Algorithm::Diff;

BEGIN {
    $VERSION = '0.04';
    require XSLoader;
    XSLoader::load( __PACKAGE__, $VERSION );

    my $code = do {
        open my $fh, '<', $INC{'Algorithm/Diff.pm'}
          or die "Cannot read $INC{'Algorithm/Diff.pm'}: $!";
        local $/;
        <$fh>;
    };

    {
        no warnings;
        local $@;
        $code =~ s/Algorithm::Diff/Algorithm::Diff::XS/g;
        $code =~ s/sub LCSidx/sub LCSidx_old/g;
        $code = "#line 1 " . __FILE__ . "\n$code";
        eval $code;
        die $@ if $@;
    }

    no warnings 'redefine';

    sub LCSidx {
        my $lcs = Algorithm::Diff::XS->_CREATE_;
        my ( @l, @r );
        for my $chunk ( $lcs->_LCS_(@_) ) {
            push @l, $chunk->[0];
            push @r, $chunk->[1];
        }
        return ( \@l, \@r );
    }
}

sub _line_map_ {
    my $ctx = shift;
    my %lines;
    push @{ $lines{ $_[$_] } }, $_ for 0 .. $#_;    # values MUST be SvIOK
    \%lines;
}

sub _LCS_ {
    my ( $ctx, $a, $b ) = @_;
    my ( $amin, $amax, $bmin, $bmax ) = ( 0, $#$a, 0, $#$b );

    while ( $amin <= $amax and $bmin <= $bmax and $a->[$amin] eq $b->[$bmin] ) {
        $amin++;
        $bmin++;
    }
    while ( $amin <= $amax and $bmin <= $bmax and $a->[$amax] eq $b->[$bmax] ) {
        $amax--;
        $bmax--;
    }

    my $h =
      $ctx->_line_map_( @$b[ $bmin .. $bmax ] ); # line numbers are off by $bmin

    return $amin + _core_loop_( $ctx, $a, $amin, $amax, $h ) + ( $#$a - $amax )
      unless wantarray;

    my @lcs = _core_loop_( $ctx, $a, $amin, $amax, $h );
    if ( $bmin > 0 ) {
        $_->[1] += $bmin for @lcs;               # correct line numbers
    }

    map( [ $_ => $_ ], 0 .. ( $amin - 1 ) ),
      @lcs,
      map( [ $_ => ++$bmax ], ( $amax + 1 ) .. $#$a );
}

1;

__END__

=head1 NAME

Algorithm::Diff::XS - Algorithm::Diff with XS core loop

=head1 SYNOPSIS

    # Drop-in replacement to Algorithm::Diff, but "compact_diff"
    # and C<LCSidx> will run much faster for large data sets.
    use Algorithm::Diff::XS qw( compact_diff LCSidx );

=head1 DESCRIPTION

This module is a simple re-packaging of Joe Schaefer's excellent
but not very well-known L<Algorithm::LCS> with a drop-in interface
that simply re-uses the installed version of the L<Algorithm::Diff>
module.

Note that only the C<LCSidx> function is optimized in XS at the
moment, which means only C<compact_diff> will get significantly
faster for large data sets, while C<diff> and C<sdiff> will run
in identical speed as C<Algorithm::Diff>.

=head1 BENCHMARK

                      Rate     Algorithm::Diff Algorithm::Diff::XS
Algorithm::Diff     14.7/s                  --                -98%
Algorithm::Diff::XS  806/s               5402%                  --

The benchmarking script is as below:

    my @data = ([qw/a b d/ x 50], [qw/b a d c/ x 50]);
    cmpthese( 500, {
        'Algorithm::Diff' => sub {
            Algorithm::Diff::compact_diff(@data)
        },
        'Algorithm::Diff::XS' => sub {
            Algorithm::Diff::XS::compact_diff(@data)
        },
    });

=head1 SEE ALSO

L<Algorithm::Diff>, L<Algorithm::LCS>.

=head1 AUTHORS

Audrey Tang E<lt>cpan@audreyt.orgE<gt>

=head1 COPYRIGHT

Copyright 2008 by Audrey Tang E<lt>cpan@audreyt.orgE<gt>.

Contains derived code copyrighted 2003 by Joe Schaefer,
E<lt>joe+cpan@sunstarsys.comE<gt>.

This library is free software; you can redistribute it and/or modify
it under the same terms as Perl itself.

=cut
