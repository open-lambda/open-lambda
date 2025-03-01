require '_h2ph_pre.ph';

no warnings qw(redefine misc);

if((!defined(&_STDDEF_H)  && !defined(&_STDDEF_H_)  && !defined(&_ANSI_STDDEF_H)  && !defined(&__STDDEF_H__)) || defined(&__need_wchar_t) || defined(&__need_size_t) || defined(&__need_ptrdiff_t) || defined(&__need_NULL) || defined(&__need_wint_t)) {
    if((!defined(&__need_wchar_t)  && !defined(&__need_size_t)  && !defined(&__need_ptrdiff_t)  && !defined(&__need_NULL)  && !defined(&__need_wint_t))) {
	eval 'sub _STDDEF_H () {1;}' unless defined(&_STDDEF_H);
	eval 'sub _STDDEF_H_ () {1;}' unless defined(&_STDDEF_H_);
	eval 'sub _ANSI_STDDEF_H () {1;}' unless defined(&_ANSI_STDDEF_H);
    }
    unless(defined(&__sys_stdtypes_h)) {
	if(defined(&__NetBSD__)) {
	    require 'machine/ansi.ph';
	}
	if(defined (&__FreeBSD__)) {
	    require 'sys/_types.ph';
	}
	if(defined(&__NetBSD__)) {
	    if(!defined(&_SIZE_T_)  && !defined(&_BSD_SIZE_T_)) {
		eval 'sub _SIZE_T () {1;}' unless defined(&_SIZE_T);
	    }
	    if(!defined(&_PTRDIFF_T_)  && !defined(&_BSD_PTRDIFF_T_)) {
		eval 'sub _PTRDIFF_T () {1;}' unless defined(&_PTRDIFF_T);
	    }
	    if(!defined(&_WCHAR_T_)  && !defined(&_BSD_WCHAR_T_)) {
		unless(defined(&_BSD_WCHAR_T_)) {
		    eval 'sub _WCHAR_T () {1;}' unless defined(&_WCHAR_T);
		}
	    }
	    if(defined (&__need_ptrdiff_t) || defined (&_STDDEF_H_)) {
		undef(&_PTRDIFF_T_) if defined(&_PTRDIFF_T_);
		undef(&_BSD_PTRDIFF_T_) if defined(&_BSD_PTRDIFF_T_);
	    }
	    if(defined (&__need_size_t) || defined (&_STDDEF_H_)) {
		undef(&_SIZE_T_) if defined(&_SIZE_T_);
		undef(&_BSD_SIZE_T_) if defined(&_BSD_SIZE_T_);
	    }
	    if(defined (&__need_wchar_t) || defined (&_STDDEF_H_)) {
		undef(&_WCHAR_T_) if defined(&_WCHAR_T_);
		undef(&_BSD_WCHAR_T_) if defined(&_BSD_WCHAR_T_);
	    }
	}
	if(defined (&__sequent__)  && defined (&_PTRDIFF_T_)) {
	    undef(&_PTRDIFF_T_) if defined(&_PTRDIFF_T_);
	}
	if(defined (&_TYPE_ptrdiff_t)  && (defined (&__need_ptrdiff_t) || defined (&_STDDEF_H_))) {
	    undef(&_TYPE_ptrdiff_t) if defined(&_TYPE_ptrdiff_t);
	}
	if(defined (&_TYPE_size_t)  && (defined (&__need_size_t) || defined (&_STDDEF_H_))) {
	    undef(&_TYPE_size_t) if defined(&_TYPE_size_t);
	}
	if(defined (&_TYPE_wchar_t)  && (defined (&__need_wchar_t) || defined (&_STDDEF_H_))) {
	    undef(&_TYPE_wchar_t) if defined(&_TYPE_wchar_t);
	}
	if(defined (&_STDDEF_H) || defined (&__need_ptrdiff_t)) {
	    unless(defined(&_PTRDIFF_T)) {
		unless(defined(&_T_PTRDIFF_)) {
		    unless(defined(&_T_PTRDIFF)) {
			unless(defined(&__PTRDIFF_T)) {
			    unless(defined(&_PTRDIFF_T_)) {
				unless(defined(&_BSD_PTRDIFF_T_)) {
				    unless(defined(&___int_ptrdiff_t_h)) {
					unless(defined(&_GCC_PTRDIFF_T)) {
					    unless(defined(&_PTRDIFF_T_DECLARED)) {
						eval 'sub _PTRDIFF_T () {1;}' unless defined(&_PTRDIFF_T);
						eval 'sub _T_PTRDIFF_ () {1;}' unless defined(&_T_PTRDIFF_);
						eval 'sub _T_PTRDIFF () {1;}' unless defined(&_T_PTRDIFF);
						eval 'sub __PTRDIFF_T () {1;}' unless defined(&__PTRDIFF_T);
						eval 'sub _PTRDIFF_T_ () {1;}' unless defined(&_PTRDIFF_T_);
						eval 'sub _BSD_PTRDIFF_T_ () {1;}' unless defined(&_BSD_PTRDIFF_T_);
						eval 'sub ___int_ptrdiff_t_h () {1;}' unless defined(&___int_ptrdiff_t_h);
						eval 'sub _GCC_PTRDIFF_T () {1;}' unless defined(&_GCC_PTRDIFF_T);
						eval 'sub _PTRDIFF_T_DECLARED () {1;}' unless defined(&_PTRDIFF_T_DECLARED);
						unless(defined(&__PTRDIFF_TYPE__)) {
						    eval 'sub __PTRDIFF_TYPE__ () {\'long int\';}' unless defined(&__PTRDIFF_TYPE__);
						}
					    }
					}
				    }
				}
			    }
			}
		    }
		}
	    }
	    undef(&__need_ptrdiff_t) if defined(&__need_ptrdiff_t);
	}
	if(defined (&_STDDEF_H) || defined (&__need_size_t)) {
	    unless(defined(&__size_t__)) {
		unless(defined(&__SIZE_T__)) {
		    unless(defined(&_SIZE_T)) {
			unless(defined(&_SYS_SIZE_T_H)) {
			    unless(defined(&_T_SIZE_)) {
				unless(defined(&_T_SIZE)) {
				    unless(defined(&__SIZE_T)) {
					unless(defined(&_SIZE_T_)) {
					    unless(defined(&_BSD_SIZE_T_)) {
						unless(defined(&_SIZE_T_DEFINED_)) {
						    unless(defined(&_SIZE_T_DEFINED)) {
							unless(defined(&_BSD_SIZE_T_DEFINED_)) {
							    unless(defined(&_SIZE_T_DECLARED)) {
								unless(defined(&___int_size_t_h)) {
								    unless(defined(&_GCC_SIZE_T)) {
									unless(defined(&_SIZET_)) {
									    unless(defined(&__size_t)) {
										eval 'sub __size_t__ () {1;}' unless defined(&__size_t__);
										eval 'sub __SIZE_T__ () {1;}' unless defined(&__SIZE_T__);
										eval 'sub _SIZE_T () {1;}' unless defined(&_SIZE_T);
										eval 'sub _SYS_SIZE_T_H () {1;}' unless defined(&_SYS_SIZE_T_H);
										eval 'sub _T_SIZE_ () {1;}' unless defined(&_T_SIZE_);
										eval 'sub _T_SIZE () {1;}' unless defined(&_T_SIZE);
										eval 'sub __SIZE_T () {1;}' unless defined(&__SIZE_T);
										eval 'sub _SIZE_T_ () {1;}' unless defined(&_SIZE_T_);
										eval 'sub _BSD_SIZE_T_ () {1;}' unless defined(&_BSD_SIZE_T_);
										eval 'sub _SIZE_T_DEFINED_ () {1;}' unless defined(&_SIZE_T_DEFINED_);
										eval 'sub _SIZE_T_DEFINED () {1;}' unless defined(&_SIZE_T_DEFINED);
										eval 'sub _BSD_SIZE_T_DEFINED_ () {1;}' unless defined(&_BSD_SIZE_T_DEFINED_);
										eval 'sub _SIZE_T_DECLARED () {1;}' unless defined(&_SIZE_T_DECLARED);
										eval 'sub ___int_size_t_h () {1;}' unless defined(&___int_size_t_h);
										eval 'sub _GCC_SIZE_T () {1;}' unless defined(&_GCC_SIZE_T);
										eval 'sub _SIZET_ () {1;}' unless defined(&_SIZET_);
										if(defined (&__FreeBSD__) || defined(&__DragonFly__) || defined(&__FreeBSD_kernel__) || defined(&__VMS__)) {
										} else {
										    eval 'sub __size_t () {1;}' unless defined(&__size_t);
										}
										unless(defined(&__SIZE_TYPE__)) {
										    eval 'sub __SIZE_TYPE__ () {\'long unsigned int\';}' unless defined(&__SIZE_TYPE__);
										}
										if(!(defined (&__GNUG__)  && defined )) {
										    if(defined(&__BEOS__)) {
										    }
										}
									    }
									}
								    }
								}
							    }
							}
						    }
						}
					    }
					}
				    }
				}
			    }
			}
		    }
		}
	    }
	    undef(&__need_size_t) if defined(&__need_size_t);
	}
	if(defined (&_STDDEF_H) || defined (&__need_wchar_t)) {
	    unless(defined(&__wchar_t__)) {
		unless(defined(&__WCHAR_T__)) {
		    unless(defined(&_WCHAR_T)) {
			unless(defined(&_T_WCHAR_)) {
			    unless(defined(&_T_WCHAR)) {
				unless(defined(&__WCHAR_T)) {
				    unless(defined(&_WCHAR_T_)) {
					unless(defined(&_BSD_WCHAR_T_)) {
					    unless(defined(&_BSD_WCHAR_T_DEFINED_)) {
						unless(defined(&_BSD_RUNE_T_DEFINED_)) {
						    unless(defined(&_WCHAR_T_DECLARED)) {
							unless(defined(&_WCHAR_T_DEFINED_)) {
							    unless(defined(&_WCHAR_T_DEFINED)) {
								unless(defined(&_WCHAR_T_H)) {
								    unless(defined(&___int_wchar_t_h)) {
									unless(defined(&__INT_WCHAR_T_H)) {
									    unless(defined(&_GCC_WCHAR_T)) {
										eval 'sub __wchar_t__ () {1;}' unless defined(&__wchar_t__);
										eval 'sub __WCHAR_T__ () {1;}' unless defined(&__WCHAR_T__);
										eval 'sub _WCHAR_T () {1;}' unless defined(&_WCHAR_T);
										eval 'sub _T_WCHAR_ () {1;}' unless defined(&_T_WCHAR_);
										eval 'sub _T_WCHAR () {1;}' unless defined(&_T_WCHAR);
										eval 'sub __WCHAR_T () {1;}' unless defined(&__WCHAR_T);
										eval 'sub _WCHAR_T_ () {1;}' unless defined(&_WCHAR_T_);
										eval 'sub _BSD_WCHAR_T_ () {1;}' unless defined(&_BSD_WCHAR_T_);
										eval 'sub _WCHAR_T_DEFINED_ () {1;}' unless defined(&_WCHAR_T_DEFINED_);
										eval 'sub _WCHAR_T_DEFINED () {1;}' unless defined(&_WCHAR_T_DEFINED);
										eval 'sub _WCHAR_T_H () {1;}' unless defined(&_WCHAR_T_H);
										eval 'sub ___int_wchar_t_h () {1;}' unless defined(&___int_wchar_t_h);
										eval 'sub __INT_WCHAR_T_H () {1;}' unless defined(&__INT_WCHAR_T_H);
										eval 'sub _GCC_WCHAR_T () {1;}' unless defined(&_GCC_WCHAR_T);
										eval 'sub _WCHAR_T_DECLARED () {1;}' unless defined(&_WCHAR_T_DECLARED);
										if(defined(&_BSD_WCHAR_T_)) {
										    undef(&_BSD_WCHAR_T_) if defined(&_BSD_WCHAR_T_);
										    if(defined(&_BSD_RUNE_T_)) {
											if(!defined (&_ANSI_SOURCE)  && !defined (&_POSIX_SOURCE)) {
											    eval 'sub _BSD_WCHAR_T_DEFINED_ () {1;}' unless defined(&_BSD_WCHAR_T_DEFINED_);
											    eval 'sub _BSD_RUNE_T_DEFINED_ () {1;}' unless defined(&_BSD_RUNE_T_DEFINED_);
											    if(defined (&__FreeBSD__)  && ((defined(&__FreeBSD__) ? &__FreeBSD__ : undef) < 5)) {
												undef(&_BSD_RUNE_T_) if defined(&_BSD_RUNE_T_);
											    }
											}
										    }
										}
										if(defined (&__FreeBSD__)  && ((defined(&__FreeBSD__) ? &__FreeBSD__ : undef) >= 5)) {
										    if(!defined (&_ANSI_SOURCE)  && !defined (&_POSIX_SOURCE)) {
											if((defined(&__BSD_VISIBLE) ? &__BSD_VISIBLE : undef)) {
											    unless(defined(&_RUNE_T_DECLARED)) {
												eval 'sub _RUNE_T_DECLARED () {1;}' unless defined(&_RUNE_T_DECLARED);
											    }
											}
										    }
										}
										unless(defined(&__WCHAR_TYPE__)) {
										    eval 'sub __WCHAR_TYPE__ () {\'int\';}' unless defined(&__WCHAR_TYPE__);
										}
										unless(defined(&__cplusplus)) {
										}
									    }
									}
								    }
								}
							    }
							}
						    }
						}
					    }
					}
				    }
				}
			    }
			}
		    }
		}
	    }
	    undef(&__need_wchar_t) if defined(&__need_wchar_t);
	}
	if(defined (&__need_wint_t)) {
	    unless(defined(&_WINT_T)) {
		eval 'sub _WINT_T () {1;}' unless defined(&_WINT_T);
		unless(defined(&__WINT_TYPE__)) {
		    eval 'sub __WINT_TYPE__ () {\'unsigned int\';}' unless defined(&__WINT_TYPE__);
		}
	    }
	    undef(&__need_wint_t) if defined(&__need_wint_t);
	}
	if(defined(&__NetBSD__)) {
	    if(defined(&_GCC_PTRDIFF_T_)) {
		undef(&_PTRDIFF_T_) if defined(&_PTRDIFF_T_);
		undef(&_BSD_PTRDIFF_T_) if defined(&_BSD_PTRDIFF_T_);
	    }
	    if(defined(&_GCC_SIZE_T_)) {
		undef(&_SIZE_T_) if defined(&_SIZE_T_);
		undef(&_BSD_SIZE_T_) if defined(&_BSD_SIZE_T_);
	    }
	    if(defined(&_GCC_WCHAR_T_)) {
		undef(&_WCHAR_T_) if defined(&_WCHAR_T_);
		undef(&_BSD_WCHAR_T_) if defined(&_BSD_WCHAR_T_);
	    }
	    if(defined(&_GCC_PTRDIFF_T)) {
		undef(&_PTRDIFF_T_) if defined(&_PTRDIFF_T_);
		undef(&_BSD_PTRDIFF_T_) if defined(&_BSD_PTRDIFF_T_);
	    }
	    if(defined(&_GCC_SIZE_T)) {
		undef(&_SIZE_T_) if defined(&_SIZE_T_);
		undef(&_BSD_SIZE_T_) if defined(&_BSD_SIZE_T_);
	    }
	    if(defined(&_GCC_WCHAR_T)) {
		undef(&_WCHAR_T_) if defined(&_WCHAR_T_);
		undef(&_BSD_WCHAR_T_) if defined(&_BSD_WCHAR_T_);
	    }
	}
    }
    if(defined (&_STDDEF_H) || defined (&__need_NULL)) {
	undef(&NULL) if defined(&NULL);
	if(defined(&__GNUG__)) {
	    eval 'sub NULL () { &__null;}' unless defined(&NULL);
	} else {
	    unless(defined(&__cplusplus)) {
		eval 'sub NULL () {(( &void *)0);}' unless defined(&NULL);
	    } else {
		eval 'sub NULL () {0;}' unless defined(&NULL);
	    }
	}
    }
    undef(&__need_NULL) if defined(&__need_NULL);
    if(defined(&_STDDEF_H)) {
	eval 'sub offsetof {
	    my($TYPE, $MEMBER) = @_;
    	    eval q( &__builtin_offsetof ($TYPE, $MEMBER));
	}' unless defined(&offsetof);
	if((defined (&__STDC_VERSION__)  && (defined(&__STDC_VERSION__) ? &__STDC_VERSION__ : undef) >= 201112) || (defined(&__cplusplus)  && (defined(&__cplusplus) ? &__cplusplus : undef) >= 201103)) {
	    unless(defined(&_GCC_MAX_ALIGN_T)) {
		eval 'sub _GCC_MAX_ALIGN_T () {1;}' unless defined(&_GCC_MAX_ALIGN_T);
		if(defined(&__i386__)) {
		}
	    }
	}
	if(defined(&__cplusplus)  && (defined(&__cplusplus) ? &__cplusplus : undef) >= 201103) {
	    unless(defined(&_GXX_NULLPTR_T)) {
		eval 'sub _GXX_NULLPTR_T () {1;}' unless defined(&_GXX_NULLPTR_T);
	    }
	}
    }
}
1;
