import React, { useCallback, useState } from "react"

export const Dropdown = ({
  links,
  titles,
}: {
  links: string[]
  titles: string[]
}) => {
  const [isOpen, setIsOpen] = useState(false)
  const toggleDropdown = useCallback(() => setIsOpen((isOpen) => !isOpen), [])

  return (
    <div>
      <button onClick={toggleDropdown}>Dropdown</button>
      {isOpen && (
        <ul>
          {links.map((link, i) => (
            <li key={i}>
              <a href={link}>{titles[i]}</a>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
