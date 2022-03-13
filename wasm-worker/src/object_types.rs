use std::collections::HashMap;
use std::fs::{read_dir, File};
use std::path::{Path, PathBuf};
use std::sync::Arc;

use crate::functions::FunctionManager;

use serde::{Deserialize, Serialize};

use lambda_store_utils::Configuration;
use open_lambda_protocol::{
    CollectionId, FieldId, FieldType, FunctionId, FunctionMetadata, ObjectTypeId, ROOT_OBJECT_TYPE,
};

pub struct ObjectTypeLoader {}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct FunctionMetadataInfo {
    data_access: bool,
    deterministic: bool,
    has_calls: bool,
}

/// The format in which the object types are read from disk
#[derive(Debug, Clone, Serialize, Deserialize)]
struct ObjectType {
    functions: HashMap<String, FunctionMetadataInfo>,
    fields: HashMap<String, FieldType>,
    collections: Vec<String>,
}

impl ObjectTypeLoader {
    pub fn new() -> Self {
        Self {}
    }

    pub async fn run(
        &self,
        config: &Arc<Configuration>,
        function_mgr: &Arc<FunctionManager>,
        registry_path: &str,
    ) {
        let compiler_name = format!("{}", function_mgr.get_compiler_type()).to_lowercase();
        let cache_path: PathBuf = format!("{registry_path}.{compiler_name}.cache").into();

        let directory = match read_dir(&registry_path) {
            Ok(dir) => dir,
            Err(err) => {
                panic!("Failed to open program registry at {registry_path:?}: {err}");
            }
        };

        let mut object_types: HashMap<String, ObjectType> = Default::default();

        // Special root object type
        {
            let fields = vec![(0, "children".to_string(), FieldType::List)];
            config.set_object_type("root".to_string(), ROOT_OBJECT_TYPE, vec![], fields, vec![]);
        }

        for entry in directory {
            let entry = entry.expect("Failed to read next file");
            let file_path = entry.path();

            if !entry.file_type().unwrap().is_file() {
                log::warn!("Entry {file_path:?} is not a regular file. Skipping...");
                continue;
            }

            let extension = match file_path.extension() {
                Some(ext) => ext,
                None => {
                    log::warn!("Entry {file_path:?} does not have a file extension. Skipping...");
                    continue;
                }
            };

            if extension == "wasm" {
                // will be opened later
                continue;
            }

            if extension != "type" {
                log::warn!("Entry {file_path:?} is not a type of WebAssembly file. Skipping...");
                continue;
            }

            let name: String = file_path
                .file_stem()
                .expect("Invalid file name")
                .to_str()
                .unwrap()
                .to_string();

            let reader = File::open(file_path.clone()).expect("Failed to open {file_path}");
            let type_data = match ron::de::from_reader(reader) {
                Ok(data) => data,
                Err(err) => {
                    panic!("Failed to parse object type file {file_path:?}: {err}");
                }
            };

            object_types.insert(name, type_data);
        }

        for (pos, (name, mut object_type)) in object_types.drain().enumerate() {
            log::debug!("Found object type \"{name}\"");
            let type_id = (pos + 1) as ObjectTypeId;
            let mut functions = vec![];
            let mut collections = vec![];
            let mut fields = vec![];

            for (pos, col_name) in object_type.collections.drain(..).enumerate() {
                collections.push((pos as CollectionId, col_name));
            }

            for (pos, (name, ftype)) in object_type.fields.drain().enumerate() {
                fields.push((pos as FieldId, name, ftype));
            }

            for (pos, (fn_name, metadata)) in object_type.functions.drain().enumerate() {
                log::debug!("Found function \"{name}::{fn_name}\"");
                let func_id = pos as FunctionId;

                functions.push((
                    func_id,
                    FunctionMetadata {
                        name: fn_name,
                        data_access: metadata.data_access,
                        has_calls: metadata.has_calls,
                        deterministic: metadata.deterministic,
                    },
                ))
            }

            {
                let file_name = format!("{registry_path}/{name}.wasm");
                let function_mgr = function_mgr.clone();

                FunctionManager::load_object_functions(
                    function_mgr,
                    type_id,
                    Path::new(&file_name).to_path_buf(),
                    cache_path.clone(),
                )
                .await;
            }

            config.set_object_type(name, type_id, collections, fields, functions);
        }
    }
}
