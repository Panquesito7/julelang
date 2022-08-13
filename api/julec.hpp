// Copyright 2022 The Jule Programming Language.
// Use of this source code is governed by a BSD 3-Clause
// license that can be found in the LICENSE file.

#ifndef __JULEC_HPP
#define __JULEC_HPP

#if defined(WIN32) || defined(_WIN32) || defined(__WIN32__) || defined(__NT__)
#ifndef _WINDOWS
#define _WINDOWS
#endif // #ifndef _WINDOWS
#endif // #if ...


#include <iostream>
#include <cstring>
#include <string>
#include <sstream>
#include <functional>
#include <vector>
#include <map>
#include <thread>
#include <typeinfo>
#include <any>
#ifdef _WINDOWS
#include <codecvt>
#include <windows.h>
#include <fcntl.h>
#endif // #ifdef _WINDOWS


#define __JULEC_EXIT_PANIC 2
#define _CONCAT(_A, _B) _A ## _B
#define CONCAT(_A, _B) _CONCAT(_A, _B)
#define JULEC_ID(_Identifier) CONCAT(_, _Identifier)
#define nil nullptr
#define CO(_Expr) std::thread{[&](void) mutable -> void { _Expr; }}.detach()

// Libraries uses this function for throw panic.
void JULEC_ID(panic)(const char *_Message);

#include "typedef.hpp"
#include "trait.hpp"
#include "slice.hpp"
#include "array.hpp"
#include "map.hpp"
#include "utf8.hpp"
#include "str.hpp"
#include "any.hpp"
#include "ptr.hpp"
#include "func.hpp"
#include "defer.hpp"
#include "builtin.hpp"

// Declarations

// Moves to heap pointer types.
template<typename T>
inline ptr<T> &__julec_must_heap(const ptr<T> &_Ptr) noexcept;
// Interface for ignore compilation errors.
template<typename T>
inline T __julec_must_heap(const T &_Obj) noexcept;
template <typename _Enum_t, typename _Index_t, typename _Item_t>
static inline void foreach(const _Enum_t _Enum,
                           const std::function<void(_Index_t, _Item_t)> _Body);
template <typename _Enum_t, typename _Index_t>
static inline void foreach(const _Enum_t _Enum,
                           const std::function<void(_Index_t)> _Body);
template <typename _Key_t, typename _Value_t>
static inline void foreach(const map<_Key_t, _Value_t> _Map,
                           const std::function<void(_Key_t)> _Body);
template <typename _Key_t, typename _Value_t>
static inline void foreach(const map<_Key_t, _Value_t> _Map,
                           const std::function<void(_Key_t, _Value_t)> _Body);
template<typename Type, unsigned N, unsigned Last>
struct tuple_ostream;
template<typename Type, unsigned N>
struct tuple_ostream<Type, N, N>;
template<typename... Types>
std::ostream &operator<<(std::ostream &_Stream,
                         const std::tuple<Types...> &_Tuple);
template<typename _Func_t, typename _Tuple_t, size_t ... _I_t>
inline auto tuple_as_args(const func<_Func_t> &_Function,
                          const _Tuple_t _Tuple,
                          const std::index_sequence<_I_t ...>);
template<typename _Func_t, typename _Tuple_t>
inline auto tuple_as_args(const func<_Func_t> &_Function, const _Tuple_t _Tuple);
std::ostream &operator<<(std::ostream &_Stream, const i8_julet &_Src);
std::ostream &operator<<(std::ostream &_Stream, const u8_julet &_Src);
template<typename _Obj_t>
str_julet tostr(const _Obj_t &_Obj) noexcept;
void x_terminate_handler(void) noexcept;
// Entry point function of generated Jule code, generates by JuleC.
void JULEC_ID(main)(void);
// Package initializer caller function, generates by JuleC.
void _julec___call_initializers(void);
int main(void);

// Definitions

template<typename T>
inline ptr<T> &__julec_must_heap(const ptr<T> &_Ptr) noexcept
{ return ((ptr<T>&)(_Ptr)).__must_heap(); }

template<typename T>
inline T __julec_must_heap(const T &_Obj) noexcept { return _Obj; }

template <typename _Enum_t, typename _Index_t, typename _Item_t>
static inline void foreach(const _Enum_t _Enum,
                           const std::function<void(_Index_t, _Item_t)> _Body) {
    _Index_t _index{0};
    for (auto _item: _Enum) { _Body(_index++, _item); }
}

template <typename _Enum_t, typename _Index_t>
static inline void foreach(const _Enum_t _Enum,
                           const std::function<void(_Index_t)> _Body) {
    _Index_t _index{0};
    for (auto begin = _Enum.begin(), end = _Enum.end(); begin < end; ++begin)
    { _Body(_index++); }
}

template <typename _Key_t, typename _Value_t>
static inline void foreach(const map<_Key_t, _Value_t> _Map,
                           const std::function<void(_Key_t)> _Body) {
    for (const auto _pair: _Map) { _Body(_pair.first); }
}

template <typename _Key_t, typename _Value_t>
static inline void foreach(const map<_Key_t, _Value_t> _Map,
                           const std::function<void(_Key_t, _Value_t)> _Body) {
    for (const auto _pair: _Map) { _Body(_pair.first, _pair.second); }
}

template<typename Type, unsigned N, unsigned Last>
struct tuple_ostream {
    static void arrow(std::ostream &_Stream, const Type &_Type) {
        _Stream << std::get<N>(_Type) << ", ";
        tuple_ostream<Type, N + 1, Last>::arrow(_Stream, _Type);
    }
};

template<typename Type, unsigned N>
struct tuple_ostream<Type, N, N> {
    static void arrow(std::ostream &_Stream, const Type &_Type)
    { _Stream << std::get<N>(_Type); }
};

template<typename... Types>
std::ostream &operator<<(std::ostream &_Stream,
                         const std::tuple<Types...> &_Tuple) {
    _Stream << '(';
    tuple_ostream<std::tuple<Types...>, 0, sizeof...(Types)-1>::arrow(_Stream, _Tuple);
    _Stream << ')';
    return _Stream;
}

template<typename _Func_t, typename _Tuple_t, size_t ... _I_t>
inline auto tuple_as_args(const func<_Func_t> &_Function,
                          const _Tuple_t _Tuple,
                          const std::index_sequence<_I_t ...>)
{ return _Function._buffer(std::get<_I_t>(_Tuple) ...); }

template<typename _Func_t, typename _Tuple_t>
inline auto tuple_as_args(const func<_Func_t> &_Function, const _Tuple_t _Tuple) {
    static constexpr auto _size{std::tuple_size<_Tuple_t>::value};
    return tuple_as_args(_Function, _Tuple, std::make_index_sequence<_size>{});
}

std::ostream &operator<<(std::ostream &_Stream, const i8_julet &_Src)
{ return _Stream << (i32_julet)(_Src); }

std::ostream &operator<<(std::ostream &_Stream, const u8_julet &_Src)
{ return _Stream << (i32_julet)(_Src); }

template<typename _Obj_t>
str_julet tostr(const _Obj_t &_Obj) noexcept {
    std::stringstream _stream;
    _stream << _Obj;
    return str_julet{_stream.str()};
}

void x_terminate_handler(void) noexcept {
    try { std::rethrow_exception(std::current_exception()); }
    catch (trait<JULEC_ID(Error)> _error) {
        std::cout << "panic: " << _error.get().error() << std::endl;
        std::exit(__JULEC_EXIT_PANIC);
    }
}

inline void JULEC_ID(panic)(const char *_Message) {
    struct panic_error: public JULEC_ID(Error) {
        const char *_message;
        str_julet error(void) { return this->_message; }
    };
    struct panic_error _error;
    _error._message = _Message;
    JULEC_ID(panic)(_error);
}

int main(void) {
    std::set_terminate(&x_terminate_handler);
    std::cout << std::boolalpha;
#ifdef _WINDOWS
    // Windows needs little spell for UTF-8
    SetConsoleOutputCP(CP_UTF8);
    _setmode(_fileno(stdin), 0x00020000);
#endif

    _julec___call_initializers();
    JULEC_ID(main());

    return EXIT_SUCCESS;
}

#endif // #ifndef __JULEC_HPP