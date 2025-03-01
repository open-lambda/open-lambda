require '_h2ph_pre.ph';

no warnings qw(redefine misc);

require 'bits/wordsize.ph';
require 'bits/timesize.ph';
if(defined (&_TIME_BITS)) {
    if((defined(&_TIME_BITS) ? &_TIME_BITS : undef) == 64) {
	if(! defined (&_FILE_OFFSET_BITS) || (defined(&_FILE_OFFSET_BITS) ? &_FILE_OFFSET_BITS : undef) != 64) {
	    die("_TIME_BITS=64 is allowed only with _FILE_OFFSET_BITS=64");
	}
 elsif((defined(&__TIMESIZE) ? &__TIMESIZE : undef) == 32) {
	    eval 'sub __USE_TIME_BITS64 () {1;}' unless defined(&__USE_TIME_BITS64);
	}
    }
 elsif((defined(&_TIME_BITS) ? &_TIME_BITS : undef) == 32) {
	if((defined(&__TIMESIZE) ? &__TIMESIZE : undef) > 32) {
	    die("_TIME_BITS=32 is not compatible with __TIMESIZE > 32");
	}
    } else {
	die("Invalid\ _TIME_BITS\ value\ \(can\ only\ be\ 32\ or\ 64\-bit\)");
    }
}
1;
