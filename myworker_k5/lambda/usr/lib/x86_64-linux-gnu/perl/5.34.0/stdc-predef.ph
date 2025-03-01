require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_STDC_PREDEF_H)) {
    eval 'sub _STDC_PREDEF_H () {1;}' unless defined(&_STDC_PREDEF_H);
    if(defined(&__GCC_IEC_559)) {
	if((defined(&__GCC_IEC_559) ? &__GCC_IEC_559 : undef) > 0) {
	    eval 'sub __STDC_IEC_559__ () {1;}' unless defined(&__STDC_IEC_559__);
	    eval 'sub __STDC_IEC_60559_BFP__ () {201404;}' unless defined(&__STDC_IEC_60559_BFP__);
	}
    } else {
	eval 'sub __STDC_IEC_559__ () {1;}' unless defined(&__STDC_IEC_559__);
	eval 'sub __STDC_IEC_60559_BFP__ () {201404;}' unless defined(&__STDC_IEC_60559_BFP__);
    }
    if(defined(&__GCC_IEC_559_COMPLEX)) {
	if((defined(&__GCC_IEC_559_COMPLEX) ? &__GCC_IEC_559_COMPLEX : undef) > 0) {
	    eval 'sub __STDC_IEC_559_COMPLEX__ () {1;}' unless defined(&__STDC_IEC_559_COMPLEX__);
	    eval 'sub __STDC_IEC_60559_COMPLEX__ () {201404;}' unless defined(&__STDC_IEC_60559_COMPLEX__);
	}
    } else {
	eval 'sub __STDC_IEC_559_COMPLEX__ () {1;}' unless defined(&__STDC_IEC_559_COMPLEX__);
	eval 'sub __STDC_IEC_60559_COMPLEX__ () {201404;}' unless defined(&__STDC_IEC_60559_COMPLEX__);
    }
    eval 'sub __STDC_ISO_10646__ () {201706;}' unless defined(&__STDC_ISO_10646__);
}
1;
