require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&__siginfo_t_defined)) {
    eval 'sub __siginfo_t_defined () {1;}' unless defined(&__siginfo_t_defined);
    require 'bits/wordsize.ph';
    require 'bits/types.ph';
    require 'bits/types/__sigval_t.ph';
    eval 'sub __SI_MAX_SIZE () {128;}' unless defined(&__SI_MAX_SIZE);
    if((defined(&__WORDSIZE) ? &__WORDSIZE : undef) == 64) {
	eval 'sub __SI_PAD_SIZE () {(( &__SI_MAX_SIZE / $sizeof{\'int\'}) - 4);}' unless defined(&__SI_PAD_SIZE);
    } else {
	eval 'sub __SI_PAD_SIZE () {(( &__SI_MAX_SIZE / $sizeof{\'int\'}) - 3);}' unless defined(&__SI_PAD_SIZE);
    }
    require 'bits/siginfo-arch.ph';
    unless(defined(&__SI_ALIGNMENT)) {
	eval 'sub __SI_ALIGNMENT () {1;}' unless defined(&__SI_ALIGNMENT);
    }
    unless(defined(&__SI_BAND_TYPE)) {
	eval 'sub __SI_BAND_TYPE () {\'long int\';}' unless defined(&__SI_BAND_TYPE);
    }
    unless(defined(&__SI_CLOCK_T)) {
	eval 'sub __SI_CLOCK_T () { &__clock_t;}' unless defined(&__SI_CLOCK_T);
    }
    unless(defined(&__SI_ERRNO_THEN_CODE)) {
	eval 'sub __SI_ERRNO_THEN_CODE () {1;}' unless defined(&__SI_ERRNO_THEN_CODE);
    }
    unless(defined(&__SI_HAVE_SIGSYS)) {
	eval 'sub __SI_HAVE_SIGSYS () {1;}' unless defined(&__SI_HAVE_SIGSYS);
    }
    unless(defined(&__SI_SIGFAULT_ADDL)) {
	eval 'sub __SI_SIGFAULT_ADDL () {1;}' unless defined(&__SI_SIGFAULT_ADDL);
    }
    if((defined(&__SI_ERRNO_THEN_CODE) ? &__SI_ERRNO_THEN_CODE : undef)) {
    } else {
    }
    if((defined(&__WORDSIZE) ? &__WORDSIZE : undef) == 64) {
    }
    if((defined(&__SI_HAVE_SIGSYS) ? &__SI_HAVE_SIGSYS : undef)) {
    }
    eval 'sub si_pid () { ($_sifields->{_kill}->{si_pid});}' unless defined(&si_pid);
    eval 'sub si_uid () { ($_sifields->{_kill}->{si_uid});}' unless defined(&si_uid);
    eval 'sub si_timerid () { ($_sifields->{_timer}->{si_tid});}' unless defined(&si_timerid);
    eval 'sub si_overrun () { ($_sifields->{_timer}->{si_overrun});}' unless defined(&si_overrun);
    eval 'sub si_status () { ($_sifields->{_sigchld}->{si_status});}' unless defined(&si_status);
    eval 'sub si_utime () { ($_sifields->{_sigchld}->{si_utime});}' unless defined(&si_utime);
    eval 'sub si_stime () { ($_sifields->{_sigchld}->{si_stime});}' unless defined(&si_stime);
    eval 'sub si_value () { ($_sifields->{_rt}->{si_sigval});}' unless defined(&si_value);
    eval 'sub si_int () { ($_sifields->{_rt}->{si_sigval}->{sival_int});}' unless defined(&si_int);
    eval 'sub si_ptr () { ($_sifields->{_rt}->{si_sigval}->{sival_ptr});}' unless defined(&si_ptr);
    eval 'sub si_addr () { ($_sifields->{_sigfault}->{si_addr});}' unless defined(&si_addr);
    eval 'sub si_addr_lsb () { ($_sifields->{_sigfault}->{si_addr_lsb});}' unless defined(&si_addr_lsb);
    eval 'sub si_lower () { ($_sifields->{_sigfault}->{_bounds}->{_addr_bnd}->{_lower});}' unless defined(&si_lower);
    eval 'sub si_upper () { ($_sifields->{_sigfault}->{_bounds}->{_addr_bnd}->{_upper});}' unless defined(&si_upper);
    eval 'sub si_pkey () { ($_sifields->{_sigfault}->{_bounds}->{_pkey});}' unless defined(&si_pkey);
    eval 'sub si_band () { ($_sifields->{_sigpoll}->{si_band});}' unless defined(&si_band);
    eval 'sub si_fd () { ($_sifields->{_sigpoll}->{si_fd});}' unless defined(&si_fd);
    if((defined(&__SI_HAVE_SIGSYS) ? &__SI_HAVE_SIGSYS : undef)) {
	eval 'sub si_call_addr () { ($_sifields->{_sigsys}->{_call_addr});}' unless defined(&si_call_addr);
	eval 'sub si_syscall () { ($_sifields->{_sigsys}->{_syscall});}' unless defined(&si_syscall);
	eval 'sub si_arch () { ($_sifields->{_sigsys}->{_arch});}' unless defined(&si_arch);
    }
}
1;
