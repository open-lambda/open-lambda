require '_h2ph_pre.ph';

no warnings qw(redefine misc);

unless(defined(&_SYS_TYPES_H)) {
    eval 'sub _SYS_TYPES_H () {1;}' unless defined(&_SYS_TYPES_H);
    require 'features.ph';
    require 'bits/types.ph';
    if(defined(&__USE_MISC)) {
	unless(defined(&__u_char_defined)) {
	    eval 'sub __u_char_defined () {1;}' unless defined(&__u_char_defined);
	}
    }
    unless(defined(&__ino_t_defined)) {
	unless(defined(&__USE_FILE_OFFSET64)) {
	} else {
	}
	eval 'sub __ino_t_defined () {1;}' unless defined(&__ino_t_defined);
    }
    if(defined (&__USE_LARGEFILE64)  && !defined (&__ino64_t_defined)) {
	eval 'sub __ino64_t_defined () {1;}' unless defined(&__ino64_t_defined);
    }
    unless(defined(&__dev_t_defined)) {
	eval 'sub __dev_t_defined () {1;}' unless defined(&__dev_t_defined);
    }
    unless(defined(&__gid_t_defined)) {
	eval 'sub __gid_t_defined () {1;}' unless defined(&__gid_t_defined);
    }
    unless(defined(&__mode_t_defined)) {
	eval 'sub __mode_t_defined () {1;}' unless defined(&__mode_t_defined);
    }
    unless(defined(&__nlink_t_defined)) {
	eval 'sub __nlink_t_defined () {1;}' unless defined(&__nlink_t_defined);
    }
    unless(defined(&__uid_t_defined)) {
	eval 'sub __uid_t_defined () {1;}' unless defined(&__uid_t_defined);
    }
    unless(defined(&__off_t_defined)) {
	unless(defined(&__USE_FILE_OFFSET64)) {
	} else {
	}
	eval 'sub __off_t_defined () {1;}' unless defined(&__off_t_defined);
    }
    if(defined (&__USE_LARGEFILE64)  && !defined (&__off64_t_defined)) {
	eval 'sub __off64_t_defined () {1;}' unless defined(&__off64_t_defined);
    }
    unless(defined(&__pid_t_defined)) {
	eval 'sub __pid_t_defined () {1;}' unless defined(&__pid_t_defined);
    }
    if((defined (&__USE_XOPEN) || defined (&__USE_XOPEN2K8))  && !defined (&__id_t_defined)) {
	eval 'sub __id_t_defined () {1;}' unless defined(&__id_t_defined);
    }
    unless(defined(&__ssize_t_defined)) {
	eval 'sub __ssize_t_defined () {1;}' unless defined(&__ssize_t_defined);
    }
    if(defined(&__USE_MISC)) {
	unless(defined(&__daddr_t_defined)) {
	    eval 'sub __daddr_t_defined () {1;}' unless defined(&__daddr_t_defined);
	}
    }
    if((defined (&__USE_MISC) || defined (&__USE_XOPEN))  && !defined (&__key_t_defined)) {
	eval 'sub __key_t_defined () {1;}' unless defined(&__key_t_defined);
    }
    if(defined (&__USE_XOPEN) || defined (&__USE_XOPEN2K8)) {
	require 'bits/types/clock_t.ph';
    }
    require 'bits/types/clockid_t.ph';
    require 'bits/types/time_t.ph';
    require 'bits/types/timer_t.ph';
    if(defined(&__USE_XOPEN)) {
	unless(defined(&__useconds_t_defined)) {
	    eval 'sub __useconds_t_defined () {1;}' unless defined(&__useconds_t_defined);
	}
	unless(defined(&__suseconds_t_defined)) {
	    eval 'sub __suseconds_t_defined () {1;}' unless defined(&__suseconds_t_defined);
	}
    }
    eval 'sub __need_size_t () {1;}' unless defined(&__need_size_t);
    require 'stddef.ph';
    if(defined(&__USE_MISC)) {
    }
    require 'bits/stdint-intn.ph';
    if( &__GNUC_PREREQ (2, 7)) {
    } else {
    }
    eval 'sub __BIT_TYPES_DEFINED__ () {1;}' unless defined(&__BIT_TYPES_DEFINED__);
    if(defined(&__USE_MISC)) {
	require 'endian.ph';
	require 'sys/select.ph';
    }
    if((defined (&__USE_UNIX98) || defined (&__USE_XOPEN2K8))  && !defined (&__blksize_t_defined)) {
	eval 'sub __blksize_t_defined () {1;}' unless defined(&__blksize_t_defined);
    }
    unless(defined(&__USE_FILE_OFFSET64)) {
	unless(defined(&__blkcnt_t_defined)) {
	    eval 'sub __blkcnt_t_defined () {1;}' unless defined(&__blkcnt_t_defined);
	}
	unless(defined(&__fsblkcnt_t_defined)) {
	    eval 'sub __fsblkcnt_t_defined () {1;}' unless defined(&__fsblkcnt_t_defined);
	}
	unless(defined(&__fsfilcnt_t_defined)) {
	    eval 'sub __fsfilcnt_t_defined () {1;}' unless defined(&__fsfilcnt_t_defined);
	}
    } else {
	unless(defined(&__blkcnt_t_defined)) {
	    eval 'sub __blkcnt_t_defined () {1;}' unless defined(&__blkcnt_t_defined);
	}
	unless(defined(&__fsblkcnt_t_defined)) {
	    eval 'sub __fsblkcnt_t_defined () {1;}' unless defined(&__fsblkcnt_t_defined);
	}
	unless(defined(&__fsfilcnt_t_defined)) {
	    eval 'sub __fsfilcnt_t_defined () {1;}' unless defined(&__fsfilcnt_t_defined);
	}
    }
    if(defined(&__USE_LARGEFILE64)) {
    }
    if(defined (&__USE_POSIX199506) || defined (&__USE_UNIX98)) {
	require 'bits/pthreadtypes.ph';
    }
}
1;
