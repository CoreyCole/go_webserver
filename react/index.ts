import { createRoot } from "react-dom/client"
import { Dropdown } from "./components"

export function renderDropdown(links: string[], titles: string[]) {
  const dropdownRoot = document.getElementById("react-dropdown")
  if (!dropdownRoot) {
    throw new Error("Could not find element with id react-dropdown")
  }
  const reactRoot = createRoot(dropdownRoot)
  reactRoot.render(Dropdown(links, titles))
}
