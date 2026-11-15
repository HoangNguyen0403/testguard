package probe

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

/**
 * Don't break on apostrophes in KDoc: it's a {@code Map<String, List<Int>>} with ( and [.
 */
class LexValidTest {
    // it's fine: unbalanced ( in a line comment
    private val quote = '\''
    private val dq = '"'
    private val brace = '}'
    private val unicode = '\u007B'
    private val escaped = "say \"hi\" \\ \$notTemplate ${'$'} }"
    private val raw = """
        raw " quote, ""double"", $dollar ${listOf(")").size} and } brace \
    """.trimIndent()
    private val fourQuotes = """ends with quote""""

    @Test
    fun l23CharsAndStrings() {
        assertEquals('\'', quote)
        assertEquals("}", brace.toString())
        assertTrue(raw.contains("}"))
    }

    @Test
    fun l30LabelsLambdasAndGenerics() {
        val m: Map<String, List<Int>> = mapOf("a" to listOf(1, 2))
        m.forEach { (k, v) ->
            v.forEach inner@{ n ->
                if (n < 0) return@inner
                assertTrue(k.length < 2 && n > 0)
            }
        }
    }

    @Test
    fun `l41 name with ' apostrophe and ( paren`() {
        val s = "${if (dq == '"') "q" else "n"}${escaped.length}"
        assertEquals("q", s.take(1))
    }

    @Test
    fun l47WhenExpression() {
        val r = when (val c = brace) {
            '}' -> "close"
            '{' -> "open"
            else -> "other $c"
        }
        assertEquals("close", r)
    }
}
