<div align="center">
<p>
    <img width="150" src="https://raw.githubusercontent.com/jule-lang/resources/main/jule_icon.svg?sanitize=true">
</p>
<h2>The Jule Programming Language</h2>

[Website](https://jule-lang.github.io/website/) |
[Documentations](https://jule-lang.github.io/website/pages/docs.html) |
[Contributing](https://jule-lang.github.io/website/pages/contributing.html)

</strong>
</div>

<h2 id="key-features">Key Features</h2>

+ Simplicity and maintainability
+ Fast and scalable development
+ Performance-critical software
+ Memory safety
+ Immutability by default
+ As efficient and performance as C++
+ High C++ interoperability

<h2 id="motivation">Motivation</h2>

Jule is a statically typed compiled programming language designed for system development, building maintainable and reliable software.
The purpose of Jule is to keep functionality high while maintaining a simple form and readability.
It guarantees memory safety and does not contain undefined behavior.
It also has a reference compiler with obsessions that encourage developers to build safe software.

<img src="./docs/images/quicksort.png"/>


<h2 id="memory-safety">Memory Safety and Management</h2>
The memory safety and the memory management is a major challenge in the C and C++ or similar programming languages.
Jule has a reference-based memory management design to solve these issues.
Jule guarantees memory safety and uses reference counting for memory management.
An account-allocation is automatically released as soon as the reference count reaches zero.
Please read the <a href="https://jule.dev/pages/docs.html?page=memory-memory-management">memory management documentations</a> for more information about reference-counting approach of Jule.
<br><br>
Variables are immutable by default, and each variable is encouraged to be initialized at declaration.
Safe Jule performs bounds checking and nil (aka null) checking.
It is committed to no undefined behavior.
Unsafe behaviors are encouraged to be done deliberately with unsafe scopes.
Please read the <a href="https://jule.dev/pages/docs.html?page=unsafe-jule">Unsafe Jule documentations</a> for more information about of Unsafe Jule.

<h2 id="cpp-interoperability">C++ Interoperability</h2>
Jule is designed to be interoperable with C++.
A C++ header file dependency can be added to the Jule code and its functions can be linked.
It's pretty easy to write C++ code that is compatible with the Jule code compiled by the compiler.
JuleC keeps all the C++ code it uses for Jule in its <a href="https://github.com/jule-lang/jule/tree/main/api">api</a> directory.
With the help of this API, it is very easy to write C++ codes that can be fully integrated into Jule.
<ol></ol> <!-- for space -->
<img src="./docs/images/cpp_interop.png"/>

<h2 id="what-is-julec">What is JuleC?</h2>
JuleC is the name of the reference compiler for the Jule programming language.
It is the original compiler of the Jule programming language.
The features that JuleC has is a representation of the official and must-have features of the Jule programming language.
This is sort of a standard for the Jule programming language and represents the minimum competency that unofficial compilers should have.

<h2 id="future-changes">Future Changes</h2>
JuleC is in early development and currently iy can only be built from source.
However, despite being in the early development stage, many algorithms (<a href="https://github.com/jule-lang/jule/tree/main/std">see the standard library</a>) can be successfully implemented.
It is planned to rewrite the compiler with Jule after reference compiler and standard library reaches sufficient maturity.
JuleC has or is very close to many of the things Jule was intended to have, such as memory safety, properties, structures with methods and generics.
<br><br>
A release is not expected until JuleC itself is developed with the Jule programming language.
The syntax and language design of the Jule programming language has emerged and is not expected to undergo major changes.
When the reference compiler is rewritten with Jule, it is thought that AST, Lexer and some packages will be included in the standard library.
This will be a change that will cause the official compiler's project structure to be rebuilt.
The reference compiler will probably use the standard library a lot.
This will also allow developers to quickly develop tools for the language by leveraging Jule's standard library.
<br><br>
There is an idea to include a package manager in JuleC as well, although it doesn't have it yet.
Jule's modern understanding of language and convenience suggests that there should be a package manager that comes with the compiler.
This package manager will provide management of non-standard library packages developed and published by the community.
Jule's standard library only gets updates with compiler releases.
<br><br>
The language and standard library will continue to evolve and change in the future but JuleC will guarantee stability since its first stable release.
Some packages of the standard library
(<a href="https://github.com/jule-lang/jule/tree/main/std/math">std::math</a>,
<a href="https://github.com/jule-lang/jule/tree/main/std/conv">std::conv</a>,
<a href="https://github.com/jule-lang/jule/tree/main/std/unicode/utf8">std::unicode::utf8</a>
or etc.) are almost complete and are not expected to undergo major changes.

<h2 id="documentations">Documentations</h2>

All documentation about Jule and JuleC is on the website. <br>
[See Documentations](https://jule-lang.github.io/website/pages/docs.html)
<br><br>
To contribute website, documentations or something else, please use the <a href="https://github.com/jule-lang/website">website repository</a>.

<h2 id="os-support">C++ Compiler Support</h2>
JuleC commits that the codes it produces can be successfully compiled by the GNU Compiler Collection and Clang.
Likewise, JuleC undertakes that the <a href="https://github.com/jule-lang/jule/tree/main/api">API</a> it has and openly offers can be compiled with these compilers. While this preference is entirely left to the developers, JuleC has primarily embraced generating Clang compatible code, so it is recommended that Clang be the primary choice.
<br><br>
Even if you can compile code generated outside of these compilers with a different compiler, it is not under official support and there is no commitment that the code will be compiled.
<br><br>
JuleC currently only promises to enable compiling generated code with these compilers and does not come with a compiler package in its own right.
However, starting from the first stable release, JuleC is planned to come with a compiler package.
It is highly likely that this compiler package is Clang.

<h2 id="os-support">Platform Support</h2>
Jule supports multiple platforms.
It supports development on i386, amd64 architectures on Windows, Linux and Darwin platforms.
JuleC undertakes that the code and standard library it produces will be compatible with all these platforms.
All supported platforms by JuleC are documented in the <a href="https://jule.dev/pages/docs.html?page=compiler-platform-support">Platform Support documentations</a>.

<h2 id="building-project">Building Project</h2>

> [Website documentation](https://jule-lang.github.io/website/pages/docs.html?page=getting-started-install-from-source) for install from source.

There are scripts prepared for compiling of JuleC. <br>
These scripts are written to run from the home directory.

`build` scripts used for compile. <br>
`brun` scripts used for compile and execute if compiling is successful.

[Go to scripts directory](scripts)

JuleC aims to have a single main build file. <br>
JuleC is in development with the [Go](https://github.com/golang/go) programming language. <br>

### Building with Go Compiler

#### Windows - PowerShell
```
go build -o julec.exe -v cmd/julec/main.go
```

#### Linux - Bash
```
go build -o julec -v cmd/julec/main.go
```

Run the above command in your terminal, in the Jule project directory.

<h2 id="project-build-state">Project Build State</h2>

<table>
    <tr>
        <td><strong>Operating System</strong></td>
        <td><strong>State</strong></td>
    </tr>
    <tr>
        <td>Windows</td>
        <td>
            <a href="https://github.com/jule-lang/jule/actions/workflows/windows.yml">
                <img src="https://github.com/jule-lang/jule/actions/workflows/windows.yml/badge.svg")>
            </a>
        </td>
    </tr>
    <tr>
        <td>Ubuntu</td>
        <td>
            <a href="https://github.com/jule-lang/jule/actions/workflows/ubuntu.yml">
                <img src="https://github.com/jule-lang/jule/actions/workflows/ubuntu.yml/badge.svg")>
            </a>
        </td>
    </tr>
    <tr>
        <td>Darwin</td>
        <td>
            <a href="https://github.com/jule-lang/jule/actions/workflows/darwin.yml">
                <img src="https://github.com/jule-lang/jule/actions/workflows/darwin.yml/badge.svg")>
            </a>
        </td>
    </tr>
</table>

<h2 id="contributing">Contributing</h2>

Thanks for you want contributing to Jule!
<br>
Every contribution, big or small, to Jule is greatly appreciated.
<br><br>
The Jule project use issues for only bug reports and proposals. <br>
To contribute, please read the contribution guidelines from <a href="https://jule-lang.github.io/website/pages/contributing.html">here</a>. <br>
To discussions and questions, please use <a href="https://github.com/jule-lang/jule/discussions">discussions</a>.

<h2 id="code-of-conduct">Code of Conduct</h2>

[See Code of Conduct](https://jule-lang.github.io/website/pages/code_of_conduct.html)

<h2 id="license">License</h2>

The JuleC and standard library is distributed under the terms of the BSD 3-Clause license. <br>
[See License Details](https://jule-lang.github.io/website/pages/license.html)
