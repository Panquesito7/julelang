// Copyright 2022 The Jule Programming Language.
// Use of this source code is governed by a BSD 3-Clause
// license that can be found in the LICENSE file.

#ifndef __JULEC_BUILTIN_HPP
#define __JULEC_BUILTIN_HPP

typedef u8_julet   JULEC_ID(byte); // builtin: type byte: u8
typedef i32_julet  JULEC_ID(rune); // builtin: type rune: i32

// Declarations
struct JULEC_ID(Error);
template<typename _Obj_t>
inline void JULEC_ID(out)(const _Obj_t _Obj) noexcept;
template<typename _Obj_t>
inline void JULEC_ID(outln)(const _Obj_t _Obj) noexcept;
inline void JULEC_ID(panic)(trait<JULEC_ID(Error)> _Error);
template<typename _Item_t>
int_julet JULEC_ID(copy)(const slice<_Item_t> &_Dest,
                         const slice<_Item_t> &_Src) noexcept;
template<typename _Item_t>
slice<_Item_t> JULEC_ID(append)(const slice<_Item_t> &_Src,
                                const slice<_Item_t> &_Components) noexcept;
template<typename T>
ptr<T> JULEC_ID(new)(void) noexcept;

// Definitions

template<typename _Obj_t>
inline void JULEC_ID(out)(const _Obj_t _Obj) noexcept { std::cout <<_Obj; }

template<typename _Obj_t>
inline void JULEC_ID(outln)(const _Obj_t _Obj) noexcept {
    JULEC_ID(out)<_Obj_t>(_Obj);
    std::cout << std::endl;
}

struct JULEC_ID(Error) {
    virtual str_julet error(void) = 0;
};

inline void JULEC_ID(panic)(trait<JULEC_ID(Error)> _Error) { throw _Error; }

template<typename _Item_t>
inline slice<_Item_t> JULEC_ID(make)(const int_julet &_N) noexcept
{ return _N < 0 ? nil : slice<_Item_t>(_N); }

template<typename _Item_t>
int_julet JULEC_ID(copy)(const slice<_Item_t> &_Dest,
                         const slice<_Item_t> &_Src) noexcept {
    if (_Dest.empty() || _Src.empty()) { return 0; }
    int_julet _len;
    if (_Dest.len() > _Src.len())      { _len = _Src.len(); }
    else if (_Src.len() > _Dest.len()) { _len = _Dest.len(); }
    else                               { _len = _Src.len(); }
    for (int_julet _index{0}; _index < _len; ++_index)
    { _Dest._buffer[_index] = _Src._buffer[_index]; }
    return _len;
}

template<typename _Item_t>
slice<_Item_t> JULEC_ID(append)(const slice<_Item_t> &_Src,
                                const slice<_Item_t> &_Components) noexcept {
    const int_julet _N{_Src.len() + _Components.len()};
    slice<_Item_t> _buffer{JULEC_ID(make)<_Item_t>(_N)};
    JULEC_ID(copy)<_Item_t>(_buffer, _Src);
    for (int_julet _index{0}; _index < _Components.len(); ++_index)
    { _buffer[_Src.len()+_index] = _Components._buffer[_index]; }
    return _buffer;
}

template<typename T>
ptr<T> JULEC_ID(new)(void) noexcept {
    ptr<T> _ptr;
    _ptr._heap = new(std::nothrow) bool*{__JULEC_PTR_HEAP_TRUE};
    if (!_ptr._heap) { JULEC_ID(panic)("memory allocation failed"); }
    _ptr._ptr = new(std::nothrow) T*;
    if (!_ptr._ptr) { JULEC_ID(panic)("memory allocation failed"); }
    *_ptr._ptr = new(std::nothrow) T;
    if (!*_ptr._ptr) { JULEC_ID(panic)("memory allocation failed"); }
    _ptr._ref = new(std::nothrow) uint_julet{1};
    if (!_ptr._ref) { JULEC_ID(panic)("memory allocation failed"); }
    return _ptr;
}

#endif // #ifndef __JULEC_BUILTIN_HPP
