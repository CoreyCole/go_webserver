import React, { useState } from "react"

interface Link {
  href: string
  title: string
}

interface DropdownProps {
  links: Link[]
}

const Dropdown: React.FC<DropdownProps> = ({ links }) => {
  const [isOpen, setIsOpen] = useState(false)

  const toggleDropdown = () => setIsOpen(!isOpen)

  return (
    <div>
      <button onClick={toggleDropdown}>Links</button>
      {isOpen && (
        <ul>
          {links.map((link, index) => (
            <li key={index}>
              <a href={link.href}>{link.title}</a>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default Dropdown
