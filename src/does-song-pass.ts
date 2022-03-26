export function doesSongPass(terms: string[], songPath: string): boolean {
    if (terms.length === 0) {
        return true;
    }

    let passedOneTerm = false;

    for (let term of terms) {
        term = term.toLowerCase();

        const isExclusion = term.startsWith('!');

        if (isExclusion) {
            term = term.slice(1);
        }

        const requiredSections = term.split(/#\s*/);

        if (
            requiredSections.every((s) =>
                s.split(/,\s*/).some((w) => songPath.includes(w))
            )
        ) {
            if (isExclusion) {
                return false;
            }

            passedOneTerm = true;
        }
    }

    if (terms.every((t) => t.startsWith('!'))) {
        return true;
    }

    return passedOneTerm;
}
