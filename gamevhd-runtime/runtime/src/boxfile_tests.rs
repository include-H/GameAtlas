//! boxfile 模型单元测试（本波交付）。与 boxfile.rs 分离以保证其体积在审查上限内。
//! `super::*` 访问 boxfile 私有项。

use super::*;


    fn sample() -> BoxFile {
        BoxFile {
            game_id: "horizon-zero-dawn".into(),
            exe_relative: r"Game\HorizonZeroDawn.exe".into(),
            game_data_root: r"E:\GameData".into(),
            user_profile: r"C:\Users\Hao".into(),
            registry_hive: r"GameData\Registry\user.dat".into(),
            skip_cache_dirs: false,
            state: BoxState::Clean,
        }
    }

    fn tmp_path(name: &str) -> std::path::PathBuf {
        std::env::temp_dir().join(format!("gamevhd_boxfile_{name}_{}.json", std::process::id()))
    }

    #[test]
    fn json_round_trip() {
        let bf = sample();
        let json = bf.to_json();
        assert!(json.starts_with('{') && json.ends_with('}'));
        let back = BoxFile::from_json(&json).unwrap();
        assert_eq!(back, bf);
    }

    #[test]
    fn json_round_trip_with_escaped_paths() {
        // 反斜杠/引号路径经 from_json 解析后必须原样还原。
        let json = r#"{
            "game_id": "game\"with\"quotes",
            "exe_relative": "Game\\HorizonZeroDawn.exe",
            "game_data_root": "E:\\GameData",
            "user_profile": "C:\\Users\\Hao",
            "registry_hive": "GameData\\Registry\\user.dat",
            "state": "running"
        }"#;
        let bf = BoxFile::from_json(json).unwrap();
        assert_eq!(bf.game_id, "game\"with\"quotes");
        assert_eq!(bf.exe_relative, r"Game\HorizonZeroDawn.exe");
        assert_eq!(bf.game_data_root, r"E:\GameData");
        assert_eq!(bf.state, BoxState::Running);

        // 序列化再回环。
        let back = BoxFile::from_json(&bf.to_json()).unwrap();
        assert_eq!(back, bf);
    }

    #[test]
    fn rejects_malformed_json() {
        let cases = [
            r#"not json"#,
            r#"{"game_id": "x"}"#,                              // 缺 5 字段
            r#"{"game_id":"x","exe_relative":"e","game_data_root":"g","user_profile":"u","registry_hive":"h","state":"clean","extra":"?"}"#, // 未知字段
            r#"{"game_id":"x","exe_relative":"e","game_data_root":"g","user_profile":"u","registry_hive":"h","state":"running","game_id":"y"}"#, // 重复字段
            r#"{"game_id":"x","exe_relative":"e","game_data_root":"g","user_profile":"u","registry_hive":"h","state":"weird"}"#, // 非法状态
            r#"{"game_id": 123,"exe_relative":"e","game_data_root":"g","user_profile":"u","registry_hive":"h","state":"clean"}"#, // 值非字符串(数字)
            r#"{"game_id": "x",}"#,                             // 尾逗号
            r#"{"game_id" "x","exe_relative":"e","game_data_root":"g","user_profile":"u","registry_hive":"h","state":"clean"}"#, // 缺冒号
            r#"{"game_id": "x\q","exe_relative":"e","game_data_root":"g","user_profile":"u","registry_hive":"h","state":"clean"}"#, // 未知转义
        ];
        for (i, c) in cases.iter().enumerate() {
            assert!(
                BoxFile::from_json(c).is_err(),
                "用例 {i} 应解析失败: {c}"
            );
        }
    }

    #[test]
    fn state_machine_valid_cycle() {
        let mut bf = sample();
        assert_eq!(bf.state, BoxState::Clean);
        bf.transition(BoxState::Running).unwrap();
        bf.transition(BoxState::Cleaning).unwrap();
        bf.transition(BoxState::Clean).unwrap();
        assert_eq!(bf.state, BoxState::Clean);
    }

    #[test]
    fn state_machine_rejects_invalid_transitions() {
        let cases = [
            (BoxState::Clean, BoxState::Clean),
            (BoxState::Clean, BoxState::Cleaning),
            (BoxState::Running, BoxState::Running),
            (BoxState::Running, BoxState::Clean),
            (BoxState::Cleaning, BoxState::Cleaning),
            (BoxState::Cleaning, BoxState::Running),
        ];
        for (from, to) in cases {
            let mut bf = sample();
            bf.state = from;
            assert_eq!(
                bf.transition(to),
                Err(BoxError::InvalidTransition { from, to }),
                "{from} -> {to} 应拒绝"
            );
            assert_eq!(bf.state, from, "失败时状态不得改变");
        }
    }

    #[test]
    fn skip_cache_dirs_optional_and_round_trip() {
        // 缺省字段 → false（向后兼容旧 box.json）。
        let legacy = r#"{
            "game_id": "g",
            "exe_relative": "Game\\a.exe",
            "game_data_root": "E:\\GameData",
            "user_profile": "C:\\Users\\Hao",
            "registry_hive": "GameData\\Registry\\user.dat",
            "state": "clean"
        }"#;
        let bf = BoxFile::from_json(legacy).unwrap();
        assert!(!bf.skip_cache_dirs);

        // 布尔字面量 true 解析。
        let with_flag = r#"{
            "game_id": "g",
            "exe_relative": "Game\\a.exe",
            "game_data_root": "E:\\GameData",
            "user_profile": "C:\\Users\\Hao",
            "registry_hive": "GameData\\Registry\\user.dat",
            "skip_cache_dirs": true,
            "state": "clean"
        }"#;
        let bf2 = BoxFile::from_json(with_flag).unwrap();
        assert!(bf2.skip_cache_dirs);
        assert!(bf2.to_json().contains("\"skip_cache_dirs\": true"));
        let back = BoxFile::from_json(&bf2.to_json()).unwrap();
        assert_eq!(back, bf2);

        // 非法布尔值报错。
        let bad = with_flag.replace("true", "\"maybe\"");
        assert!(BoxFile::from_json(&bad).is_err());
    }

    #[test]
    fn atomic_save_then_load_round_trip() {
        let path = tmp_path("save");
        let tmp = format!("{}.tmp", path.display());
        let _ = fs::remove_file(&path);
        let _ = fs::remove_file(&tmp);

        let mut bf = sample();
        bf.save(&path).unwrap();
        let loaded = BoxFile::load(&path).unwrap();
        assert_eq!(loaded, bf);

        // 二次保存覆盖旧内容，且不留临时文件。
        bf.state = BoxState::Running;
        bf.save(&path).unwrap();
        let loaded2 = BoxFile::load(&path).unwrap();
        assert_eq!(loaded2.state, BoxState::Running);
        assert!(!Path::new(&tmp).exists(), "临时文件应已改名");

        let _ = fs::remove_file(&path);
    }

    #[test]
    fn load_missing_file_is_error() {
        let missing = tmp_path("missing");
        let _ = fs::remove_file(&missing);
        assert!(matches!(BoxFile::load(&missing), Err(BoxError::Io(_))));
    }
