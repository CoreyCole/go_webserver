import { createRoot } from "react-dom/client"
import { Dropdown } from "./components/dropdown"

export function renderDropdown(links: string[], titles: string[]) {
  const dropdownRoot = document.getElementById("dropdown-root")
  if (!dropdownRoot) {
    throw new Error("Could not find dropdown root element")
  }
  const reactRoot = createRoot(dropdownRoot)
  reactRoot.render(Dropdown({ links, titles }))
}
