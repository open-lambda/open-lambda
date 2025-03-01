require '_h2ph_pre.ph';

no warnings qw(redefine misc);

eval 'sub _GCC_NEXT_LIMITS_H () {1;}' unless defined(&_GCC_NEXT_LIMITS_H);
eval {
    my(@REM);
    my(%INCD) = map { $INC{$_} => 1 } (grep { $_ eq "limits.ph" } keys(%INC));
    @REM = map { "$_/limits.ph" } (grep { not exists($INCD{"$_/limits.ph"}) and -f "$_/limits.ph" } @INC);
    require "$REM[0]" if @REM;
};
warn($@) if $@;
undef(&_GCC_NEXT_LIMITS_H) if defined(&_GCC_NEXT_LIMITS_H);
1;
