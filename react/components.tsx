import React, { useState } from "react"

export const Dropdown = (links: string[], titles: string[]) => {
  const [isOpen, setIsOpen] = useState(false)
  const toggleDropdown = () => setIsOpen(!isOpen)

  return (
    <div>
      <button onClick={toggleDropdown}>Links</button>
      {isOpen && (
        <ul>
          {links.map((link, index) => (
            <li key={index}>
              <a href={link}>{titles[index]}</a>
            </li>
          ))}
        </ul>
      )}
    </div>
    // <div>
    //   {links.map((link, index) => (
    //     <div key={index}>
    //       <a href={link}>{titles[index]}</a>
    //       <br />
    //     </div>
    //   ))}
    // </div>
  )
}
