import React from "react";
import { Link } from "react-router-dom";

export default function Home() {
  return (
    <div>
      <span>hello world!!</span>
      <Link to="/about">About page</Link>
    </div>
  );
}
