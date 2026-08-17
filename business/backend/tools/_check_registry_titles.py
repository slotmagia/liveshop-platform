import re
import subprocess

out = subprocess.check_output(
    [
        "docker",
        "exec",
        "liveshop-platform-local-registry-db-1",
        "mysql",
        "-uliveshop",
        "-pliveshop-local",
        "--default-character-set=utf8mb4",
        "-N",
        "-B",
        "liveshop_registry",
        "-e",
        "SELECT releases FROM platform_registry_state",
    ],
    stderr=subprocess.DEVNULL,
)
text = out.decode("utf-8", errors="replace")
titles = re.findall(r'"title"\s*:\s*"([^"]*)"', text)
print("titles:", titles)
print("has_chinese:", any("\u4e00" <= ch <= "\u9fff" for t in titles for ch in t))
print("has_qmarks:", any("?" in t for t in titles))
