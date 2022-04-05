extern crate proc_macro;

use proc_macro::TokenStream;
use quote::quote;
use syn::{parse_macro_input, ItemFn};

#[proc_macro_attribute]
pub fn main_func(_args: TokenStream, input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as ItemFn).block;

    let expanded = quote! {
        #[ cfg(target_arch="wasm32") ]
        #[ no_mangle ]
        fn f() {
            #input
        }

        #[ cfg(target_arch="wasm32") ]
        fn main() {}

        #[ cfg(not(target_arch="wasm32")) ]
        fn main() {
            open_lambda::internal_init();
            #input
        }
    };

    TokenStream::from(expanded)
}
