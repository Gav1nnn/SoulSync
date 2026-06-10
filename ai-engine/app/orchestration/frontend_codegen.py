from __future__ import annotations

from app.schemas import AgentPlanAction, AgentStepRequest


def next_generated_frontend_action(request: AgentStepRequest) -> AgentPlanAction | None:
    if not should_generate_frontend(request):
        return None
    api = request.workspace_summary.api_candidates[0] if request.workspace_summary.api_candidates else None
    if api is None:
        return None

    paths = generated_frontend_paths(request)
    changed = set(request.changed_files)
    if paths["type"] not in changed:
        return AgentPlanAction(
            type="write_file",
            path=paths["type"],
            content=generated_type_content(api),
            reason="先写入页面使用的 TypeScript 类型。",
        )
    if paths["client"] not in changed:
        return AgentPlanAction(
            type="write_file",
            path=paths["client"],
            content=generated_api_client_content(api, paths),
            reason="写入与目标接口对应的前端 API client。",
        )
    if paths["view"] not in changed:
        return AgentPlanAction(
            type="write_file",
            path=paths["view"],
            content=generated_vue_view_content(api, paths),
            reason="生成包含加载态、错误态和空状态的 Vue 页面。",
        )
    return None


def build_generation_summary(request: AgentStepRequest) -> str:
    api = request.workspace_summary.api_candidates[0]
    changed_count = len(request.changed_files)
    return f"根据 {api.method} {api.path} 生成前端文件，已写入 {changed_count} 个文件。"


def should_generate_frontend(request: AgentStepRequest) -> bool:
    goal = request.goal.lower()
    wants_page = any(keyword in goal for keyword in ["页面", "前端", "vue", "列表", "page", "frontend"])
    return wants_page and bool(request.workspace_summary.api_candidates)


def generated_frontend_paths(request: AgentStepRequest) -> dict[str, str]:
    frontend_root = infer_frontend_root(request)
    resource = resource_name(request.workspace_summary.api_candidates[0].path)
    pascal = pascal_case(resource)
    camel = camel_case(resource)
    return {
        "resource": resource,
        "pascal": pascal,
        "camel": camel,
        "type": f"{frontend_root}/src/types/soulsync{pascal}.ts",
        "client": f"{frontend_root}/src/api/soulsync{pascal}.ts",
        "view": f"{frontend_root}/src/views/SoulSync{pascal}View.vue",
    }


def infer_frontend_root(request: AgentStepRequest) -> str:
    candidates = []
    candidates.extend(candidate.path for candidate in request.workspace_summary.frontend_entry_candidates)
    candidates.extend(candidate.path for candidate in request.workspace_summary.api_client_candidates)
    candidates.extend(candidate.path for candidate in request.workspace_summary.type_file_candidates)
    for path in candidates:
        if "/src/" in path:
            return path.split("/src/", 1)[0]
    return "frontend"


def resource_name(path: str) -> str:
    cleaned = path.strip().strip("/")
    if not cleaned:
        return "item"
    parts = [part for part in cleaned.split("/") if part and not part.startswith(":")]
    if not parts:
        return "item"
    name = parts[-1]
    if name.endswith("ies"):
        return name[:-3] + "y"
    if name.endswith("s") and len(name) > 1:
        return name[:-1]
    return name


def pascal_case(value: str) -> str:
    parts = [part for part in value.replace("-", "_").split("_") if part]
    if not parts:
        return "Item"
    return "".join(part[:1].upper() + part[1:] for part in parts)


def camel_case(value: str) -> str:
    pascal = pascal_case(value)
    return pascal[:1].lower() + pascal[1:]


def generated_type_content(api) -> str:
    pascal = pascal_case(resource_name(api.path))
    return (
        f"export type SoulSync{pascal} = {{\n"
        "  id: string | number;\n"
        "  name?: string;\n"
        "  title?: string;\n"
        "  [key: string]: unknown;\n"
        "};\n"
    )


def generated_api_client_content(api, paths: dict[str, str]) -> str:
    pascal = paths["pascal"]
    camel = paths["camel"]
    type_import = relative_import(paths["client"], paths["type"])
    return (
        f'import type {{ SoulSync{pascal} }} from "{type_import}";\n\n'
        f"export async function fetchSoulSync{pascal}List(): Promise<SoulSync{pascal}[]> {{\n"
        f'  const response = await fetch("{api.path}");\n'
        "  if (!response.ok) {\n"
        f'    throw new Error("Failed to load {camel} list");\n'
        "  }\n\n"
        f"  return (await response.json()) as SoulSync{pascal}[];\n"
        "}\n"
    )


def generated_vue_view_content(api, paths: dict[str, str]) -> str:
    pascal = paths["pascal"]
    camel = paths["camel"]
    client_import = relative_import(paths["view"], paths["client"])
    type_import = relative_import(paths["view"], paths["type"])
    return (
        '<script setup lang="ts">\n'
        'import { onMounted, ref } from "vue";\n'
        f'import {{ fetchSoulSync{pascal}List }} from "{client_import}";\n'
        f'import type {{ SoulSync{pascal} }} from "{type_import}";\n\n'
        f"const items = ref<SoulSync{pascal}[]>([]);\n"
        "const isLoading = ref(false);\n"
        "const errorMessage = ref(\"\");\n\n"
        f"async function load{pascal}List() {{\n"
        "  isLoading.value = true;\n"
        "  errorMessage.value = \"\";\n\n"
        "  try {\n"
        f"    items.value = await fetchSoulSync{pascal}List();\n"
        "  } catch (error) {\n"
        f'    errorMessage.value = error instanceof Error ? error.message : "Failed to load {camel} list.";\n'
        "  } finally {\n"
        "    isLoading.value = false;\n"
        "  }\n"
        "}\n\n"
        "onMounted(() => {\n"
        f"  void load{pascal}List();\n"
        "});\n"
        "</script>\n\n"
        "<template>\n"
        f'  <main class="soulsync-{camel}-page">\n'
        "    <header>\n"
        f"      <p>{api.method} {api.path}</p>\n"
        f"      <h1>{pascal} List</h1>\n"
        "    </header>\n\n"
        '    <p v-if="isLoading" class="state-copy">Loading...</p>\n'
        '    <p v-else-if="errorMessage" class="error-copy">{{ errorMessage }}</p>\n'
        '    <p v-else-if="!items.length" class="state-copy">No data yet.</p>\n\n'
        '    <ul v-else class="item-list">\n'
        '      <li v-for="item in items" :key="String(item.id)">\n'
        "        <strong>{{ item.name || item.title || item.id }}</strong>\n"
        "        <code>{{ item.id }}</code>\n"
        "      </li>\n"
        "    </ul>\n"
        "  </main>\n"
        "</template>\n\n"
        "<style scoped>\n"
        f".soulsync-{camel}-page {{\n"
        "  display: grid;\n"
        "  gap: 16px;\n"
        "  padding: 24px;\n"
        "}\n\n"
        "header p,\n"
        "header h1,\n"
        ".state-copy,\n"
        ".error-copy {\n"
        "  margin: 0;\n"
        "}\n\n"
        "header p {\n"
        "  color: #64748b;\n"
        "  font-size: 0.78rem;\n"
        "  font-weight: 700;\n"
        "  text-transform: uppercase;\n"
        "}\n\n"
        "header h1 {\n"
        "  margin-top: 4px;\n"
        "  color: #1f2937;\n"
        "  font-size: 1.6rem;\n"
        "}\n\n"
        ".item-list {\n"
        "  display: grid;\n"
        "  gap: 8px;\n"
        "  margin: 0;\n"
        "  padding: 0;\n"
        "  list-style: none;\n"
        "}\n\n"
        ".item-list li {\n"
        "  display: flex;\n"
        "  justify-content: space-between;\n"
        "  gap: 12px;\n"
        "  padding: 12px;\n"
        "  border: 1px solid #e5e7eb;\n"
        "  border-radius: 8px;\n"
        "}\n\n"
        ".item-list code,\n"
        ".state-copy {\n"
        "  color: #64748b;\n"
        "}\n\n"
        ".error-copy {\n"
        "  color: #b91c1c;\n"
        "}\n"
        "</style>\n"
    )


def relative_import(from_path: str, to_path: str) -> str:
    from_parts = from_path.split("/")[:-1]
    to_parts = to_path.split("/")
    while from_parts and to_parts and from_parts[0] == to_parts[0]:
        from_parts.pop(0)
        to_parts.pop(0)
    prefix = "../" * len(from_parts)
    result = prefix + "/".join(to_parts)
    if result.endswith(".ts"):
        result = result[:-3]
    if not result.startswith("."):
        result = "./" + result
    return result
